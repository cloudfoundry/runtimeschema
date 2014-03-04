package router

func NewFileServerRoutes() Routes {
	return Routes{
		{Path: "/droplet/:guid", Method: "POST", Handler: "upload_droplet"},
		{Path: "/static/", Method: "GET", Handler: "static"},
	}
}
