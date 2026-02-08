package maps

type FullName string

type Info struct {
	Age  int
	City string
}

type Target struct {
	Tags map[FullName]Info
}
