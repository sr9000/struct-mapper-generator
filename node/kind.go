package node

type DispatcherEnum int

const (
	DispatcherUnknown DispatcherEnum = iota
	DispatcherPrimitive
	DispatcherInterface
	DispatcherSlice
	DispatcherMap
	DispatcherStruct

	// DispatcherTotal is a constant that represents the total number of kinds defined
	DispatcherTotal = int(iota)
)
