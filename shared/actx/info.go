package actx

func (a *aContext) SetName(name string) {
	a.data.name = name
}

func (a *aContext) GetName() string {
	return a.data.name
}
