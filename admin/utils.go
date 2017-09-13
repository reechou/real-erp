package admin

import "github.com/qor/qor"

func IsAdmin(context *qor.Context) bool {
	if len(context.Roles) > 0 {
		for _, v := range context.Roles {
			if v == UserPermissions[USER_PERMISSION_ADMIN] {
				return true
			}
		}
	}
	return false
}
