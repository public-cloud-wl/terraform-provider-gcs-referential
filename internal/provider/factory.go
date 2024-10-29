package provider

type DataParser interface {
	Parse(data []byte) error
}

/*
func createParser(structType string) DataParser {
	switch structType {
	case "id_pool":
		return &IdPoolResourceModel{}
	case "network_request":
		return &networkRequestResourceModel{}
	default:
		return nil
	}
}
*/
