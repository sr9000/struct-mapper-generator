package maps

type Name string

type Params struct {
	Age  int
	City string
}

type Source struct {
	Tags map[Name]Params
}
