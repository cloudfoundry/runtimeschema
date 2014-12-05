package routes

import "github.com/tedsuo/rata"

const (
	FS_STATIC                 = "static"
	FS_UPLOAD_DROPLET         = "upload_droplet"
	FS_UPLOAD_BUILD_ARTIFACTS = "upload_build_artifacts"
)

var FileServerRoutes = rata.Routes{
	{Name: FS_STATIC, Method: "GET", Path: "/v1/static/"},
	{Name: FS_UPLOAD_DROPLET, Method: "POST", Path: "/v1/droplet/:guid"},
	{Name: FS_UPLOAD_BUILD_ARTIFACTS, Method: "POST", Path: "/v1/build_artifacts/:app_guid"},
}
