package pkg

import "io"

type Writer interface {
	io.ByteWriter
	io.StringWriter
	io.Writer
}

type Bencodable interface {
	WriteTo(Writer) error
}
