package tools

func init() {
	AddRegistrar(RegisterHTTPFromCatalog)
	AddRegistrar(RegisterBespokeTools)
}
