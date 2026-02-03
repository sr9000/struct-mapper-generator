package pointers

type Person struct {
	Name    string
	Age     *int
	Address *Address
}

type Address struct {
	City string
}
