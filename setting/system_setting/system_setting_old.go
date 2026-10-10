package system_setting

var ServerAddress = "http://localhost:32180"
var TaskPublicAddress = ""
var WorkerUrl = ""
var WorkerValidKey = ""
var WorkerAllowHttpImageRequestEnabled = false

func EnableWorker() bool {
	return WorkerUrl != ""
}
