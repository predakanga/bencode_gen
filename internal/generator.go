package internal

import (
	"bufio"
	"github.com/predakanga/bencode_gen/pkg"
	log "github.com/sirupsen/logrus"
	"go/ast"
	gotoken "go/token"
	"go/types"
	"golang.org/x/tools/go/packages"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var autogenRegex = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)
var pkgCfg = packages.Config{
	Mode: packages.NeedName |
		packages.NeedFiles |
		packages.NeedImports |
		packages.NeedTypes |
		packages.NeedTypesInfo,
	Fset:       gotoken.NewFileSet(),
	BuildFlags: []string{"-tags", "generate"},
}

func DoGenerate(packageNames []string, typeNames []string, mode OutputMode) {
	log.Debugf("Got packageNames: %#v", packageNames)
	// Sort the type names, to make it easier to check membership
	sort.Strings(typeNames)

	// Load the packages
	pkgs, err := packages.Load(&pkgCfg, packageNames...)
	if err != nil {
		log.Fatalf("Couldn't load packages: %v", err)
	}

	// Find our requisite interfaces
	var bencodeInterface *types.Interface
	var durationType types.Type
	for _, pkg := range pkgs {
		if thisPkg, ok := pkg.Imports["github.com/predakanga/bencode_gen/pkg"]; ok {
			bencodeInterface = thisPkg.Types.Scope().Lookup("Bencodable").Type().Underlying().(*types.Interface)
		}
		if timePkg, ok := pkg.Imports["time"]; ok {
			durationType = timePkg.Types.Scope().Lookup("Duration").Type()
		}
	}
	if bencodeInterface == nil {
		log.Fatalf("Could not locate type: github.com/predakanga/bencode_gen/pkg.Bencodable")
	}
	if durationType == nil {
		log.Fatalf("Could not locate type: time.Duration")
	}

	// Check each package for interesting types
	for _, pkg := range pkgs {
		log.Debugf("Searching package: %v (%+v)", pkg.PkgPath, pkg)
		for k, v := range pkg.TypesInfo.Defs {
			if interestingDef(k, v, typeNames) {
				pkgGen := &PackageGenerator{
					pkg:              pkg,
					bencodeInterface: bencodeInterface,
					durationType:     durationType,
				}
				pkgGen.Generate(typeNames, mode)
				break
			}
		}
	}
}

type PackageGenerator struct {
	pkg              *packages.Package
	bencodeInterface *types.Interface
	durationType     types.Type
	typeImpls        [][]token
}

func (pg *PackageGenerator) Generate(typeNames []string, mode OutputMode) {
	// Determine our output file
	if len(pg.pkg.GoFiles) == 0 {
		log.Fatalf("Could not determine package location for %v", pg.pkg)
	}
	outDir := filepath.Dir(pg.pkg.GoFiles[0])
	outPath := filepath.Join(outDir, "bencode_gen.go") // Make sure any existing file is our own
	log.Printf("Generating bencoders for %v (%v)", pg.pkg, outPath)

	// Build up our output-tree
	generatedTypes := pg.generateForTypeNames(typeNames)
	if len(generatedTypes) == 0 {
		log.Printf("Skipping %v - no valid types found", pg.pkg)
		return
	}

	// Then output if necessary
	if mode != DryRun {
		// Check that any existing file is auto-generated
		if fileExists(outPath) && mode != Overwrite {
			f, err := os.OpenFile(outPath, os.O_RDONLY, 0644)
			if err != nil {
				log.Fatalf("Could not open %v for reading: %v", outPath, err)
			}
			defer must(f.Close)
			if !autogenRegex.MatchReader(bufio.NewReader(f)) {
				log.Fatalf("%v does not seem to be auto-generated; not overwriting it. Use --force to override this.", outPath)
			}
		}

		// Do our actual output
		// Create the temporary file in our output dir, so that we know we can just rename it later
		file, err := ioutil.TempFile(outDir, ".bencode_gen.*.go")
		if err != nil {
			log.Fatalf("Could not create output file: %v", err)
		}
		tmpPath := file.Name()
		log.Debugf("Temporary output file is: %v", tmpPath)

		pg.writePackage(file)

		// Then close the output file and run it through gofmt
		log.Debugf("Running gofmt -w %v", tmpPath)
		if err := exec.Command("gofmt", "-w", tmpPath).Run(); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				log.Fatalf("Failed to run gofmt: %v", err)
			}
		}
		// Move it into place
		if err := os.Rename(tmpPath, outPath); err != nil {
			log.Fatalf("Failed to replace output file: %v", err)
		}
	}

	// And finally, inform the user
	log.Printf("Wrote %v with encoders for %v", outPath, strings.Join(generatedTypes, ", "))
}

func (pg *PackageGenerator) generateForTypeNames(names []string) (genTypes []string) {
	includeTaggedStructs := len(names) == 0 || strContains(names, "*")

	// Find out what types we want to handle and sort them (ensures deterministic output)
	var toGen DefinitionSlice
	for id, obj := range pg.pkg.TypesInfo.Defs {
		if id.Obj == nil || id.Obj.Kind != ast.Typ {
			continue
		}

		if types.Implements(obj.Type(), pg.bencodeInterface) {
			continue
		}

		if strContains(names, id.Name) || (includeTaggedStructs && structHasTag(obj.Type(), id.Name)) {
			toGen = append(toGen, Definition{id, obj})
		}
	}
	sort.Sort(toGen)

	// And do the actual generating
	for _, def := range toGen {
		genTypes = append(genTypes, def.Ident.Name)
		pg.generateForType(def.Ident, def.Object)
	}
	return
}

func (pg *PackageGenerator) generateForType(id *ast.Ident, obj types.Object) {
	log.Debugf("Generating implementation for %v", id.Name)

	// Set up a panic handler to allow *Tokens functions to panic
	defer func() {
		err := recover()
		if err != nil {
			log.Fatalf("Failed to generate type %v - %v", id.Name, err)
		}
	}()

	// Get the token list for this type
	tokens := pg.typeTokens("x", obj.Type())
	// Prepend a header containing the type name
	tokens = append([]token{{"_header", id.Name}}, tokens...)

	// And store it
	pg.typeImpls = append(pg.typeImpls, tokens)
}

func (pg *PackageGenerator) writePackage(w io.Writer) {
	// Figure out what imports we're going to need
	headerCtx := pkgHeaderContext{
		GeneratorName:    pkg.Name,
		GeneratorVersion: pkg.VersionString,
		PackageName:      pg.pkg.Name,
	}

	// Store whether a type will need to sort as well, to avoid re-iterating
	typeNeedsSort := make([]bool, len(pg.typeImpls))
	for i, typeImpl := range pg.typeImpls {
		for _, tok := range typeImpl {
			switch tok.Type {
			case "map_start":
				headerCtx.NeedSort = true
				typeNeedsSort[i] = true
			case "string", "int":
				headerCtx.NeedStrconv = true
			}
		}
	}

	// Output the header
	render("package_header", w, headerCtx)

	for i, tokens := range pg.typeImpls {
		typeName := tokens[0].Data
		render("type_start", w, typeStartContext{typeName, typeNeedsSort[i]})

		for _, tok := range mergeConstTokens(tokens[1:]) {
			switch tok.Type {
			case "const":
				if len(tok.Data) == 1 {
					render("const_byte", w, tok.Data)
				} else {
					render("const_string", w, tok.Data)
				}
			default:
				render(tok.Type, w, tok.Data)
			}
		}

		render("type_end", w, nil)
	}
}
