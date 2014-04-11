package router

const (
	FS_STATIC                   = "static"
	FS_UPLOAD_DROPLET           = "upload_droplet"
	FS_UPLOAD_BUILD_ARTIFACTS   = "upload_build_artifacts"
	FS_DOWNLOAD_BUILD_ARTIFACTS = "download_build_artifacts"
)

func NewFileServerRoutes() Routes {
	return Routes{
		{Path: "/static/", Method: "GET", Handler: FS_STATIC},
		{Path: "/droplet/:guid", Method: "POST", Handler: FS_UPLOAD_DROPLET},
		{Path: "/build_artifacts/:app_guid", Method: "PUT", Handler: FS_UPLOAD_BUILD_ARTIFACTS},
		{Path: "/build_artifacts/:app_guid", Method: "GET", Handler: FS_DOWNLOAD_BUILD_ARTIFACTS},
	}
}
