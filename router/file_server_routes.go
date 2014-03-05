package router

const (
	FS_STATIC         = "static"
	FS_UPLOAD_DROPLET = "upload_droplet"
)

func NewFileServerRoutes() Routes {
	return Routes{
		{Path: "/static/", Method: "GET", Handler: FS_STATIC},
		{Path: "/droplet/:guid", Method: "POST", Handler: FS_UPLOAD_DROPLET},
	}
}
