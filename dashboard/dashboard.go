package dashboard

type DashboardMenu struct {
	REST []RESTComponent `json:"rest"`
}

type Dashboard struct {
	Menus DashboardMenu `json:"menus"`
}

type DashboardBuilder struct {
	Menus DashboardMenu
}

func NewDashboardBuilder() *DashboardBuilder {
	return &DashboardBuilder{}
}

func (d *DashboardBuilder) AddRESTMenu(restComponent RESTComponent) *DashboardBuilder {
	d.Menus.REST = append(d.Menus.REST, restComponent)
	return d
}

func (d *DashboardBuilder) Build() *Dashboard {
	return &Dashboard{
		Menus: d.Menus,
	}
}
