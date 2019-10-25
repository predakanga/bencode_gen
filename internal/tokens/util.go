package tokens

// We can only merge consts in containers
func MergeConsts(tokens []CodeToken) (toRet []CodeToken) {
	childCount := len(tokens)
	if childCount < 2 {
		return tokens
	}

	toRet = make([]CodeToken, 0, childCount)
	buffer := ""

	for _, tok := range tokens {
		switch castTok := tok.(type) {
		case *Const:
			buffer += castTok.Data
		default:
			if buffer != "" {
				toRet = append(toRet, &Const{buffer})
				buffer = ""
			}
			if childContainer, ok := tok.(Container); ok {
				childContainer.SetContents(MergeConsts(childContainer.Contents()))
			}
			toRet = append(toRet, tok)
		}
	}

	if buffer != "" {
		toRet = append(toRet, &Const{buffer})
		buffer = ""
	}

	return
}
