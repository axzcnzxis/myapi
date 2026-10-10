package service

import (
	"github.com/axzcnzxis/myapi/setting/operation_setting"
	"github.com/axzcnzxis/myapi/setting/system_setting"
)

func GetCallbackAddress() string {
	if operation_setting.CustomCallbackAddress == "" {
		return system_setting.ServerAddress
	}
	return operation_setting.CustomCallbackAddress
}
