package menu

// Spec 内置菜单树节点（无 DB ID），path 在同级下唯一。
type Spec struct {
	Path      string
	Name      string
	Icon      string
	Sort      int
	Component string
	Redirect  string
	Hidden    bool
	AdminOnly bool
	Status    int
	Children  []Spec
}

func (s Spec) statusOrDefault() int {
	if s.Hidden {
		return 0
	}
	if s.Status != 0 {
		return s.Status
	}
	return 1
}
