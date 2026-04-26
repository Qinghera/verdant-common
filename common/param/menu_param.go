package param

// MenuParam 菜单参数
type MenuParam struct {
	ID       uint64 `json:"id"`
	ParentID uint64 `json:"parentId"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Icon     string `json:"icon"`
	Sort     int    `json:"sort"`
	Status   int    `json:"status"`
}

// MenuSortParam 菜单排序参数
type MenuSortParam struct {
	ID   uint64 `json:"id"`
	Sort int    `json:"sort"`
}
