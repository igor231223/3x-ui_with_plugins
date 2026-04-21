package service

import "github.com/mhsanaei/3x-ui/v2/plugins"

var runtimePluginManager *plugins.Manager

func SetPluginManager(m *plugins.Manager) {
	runtimePluginManager = m
}

func getPluginManager() *plugins.Manager {
	return runtimePluginManager
}

func GetPluginManager() *plugins.Manager {
	return runtimePluginManager
}
