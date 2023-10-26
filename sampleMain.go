package main

import (
	"fmt"
	logHarbour "logharbour/logHarbour"
	"time"
)

func main() {
	loggers := logHarbour.LogInit("app1", "module1", "system1")

	logHarbour.LogWrite(loggers.ActivityLogger, logHarbour.LevelInfo, "spanid1", "correlationid1", time.Now().AddDate(1, 0, 0), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id1", 1, "This is an activity logger info message", "somekey", "somevalue", "key2", "value2")
	logHarbour.LogWrite(loggers.ActivityLogger, logHarbour.LevelError, "spanid2", "correlationid2", time.Now(), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id2", 1, "This is an activity logger error message", "somekey", "somevalue",
		logHarbour.DataChg("amt", "100", "200"), logHarbour.DataChg("qty", "1", "2"))
	logHarbour.LogWrite(loggers.ActivityLogger, logHarbour.LevelError, "spanid3", "correlationid3", time.Now(), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id3", 1, "This is an activity logger error message")

	fmt.Println("----")
	logHarbour.LogWrite(loggers.DataChangeLogger, logHarbour.LevelInfo, "spanid4", "correlationid4", time.Now(), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id4", 1, "This is an DataChange logger info message", "somekey", "somevalue")

	logHarbour.LogWrite(loggers.DataChangeLogger, logHarbour.LevelInfo, "spanid5", "correlationid5", time.Now(), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id5", 1, "This is an DataChange logger info message", "somekey", "somevalue",
		logHarbour.DataChg("amt", "100", "200"), logHarbour.DataChg("qty", "1", "2"))

	logHarbour.LogWrite(loggers.DataChangeLogger, logHarbour.LevelError, "spanid6", "correlationid6", time.Now(), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id6", 1, "This is an DataChange logger error message", logHarbour.DataChg("amt", "100", "200"),
		logHarbour.DataChg("qty", "1", "2"))

	fmt.Println("----")

	logHarbour.LogWrite(loggers.DebugLogger, logHarbour.LevelDebug0, "spanid7", "correlationid7", time.Now(), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id7", 1, "This is an DEBUG0 logger message 1", "key3", "value3",
		logHarbour.DataChg("amt", "100", "200"), logHarbour.DataChg("qty", "1", "2"))

	logHarbour.LogWrite(loggers.DebugLogger, logHarbour.LevelDebug1, "spanid8", "correlationid8", time.Now(), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id8", 1, "This is an DEBUG1 logger message 2", "key3", "value3",
		logHarbour.DataChg("qty", "1", "2"))

	logHarbour.LogWrite(loggers.DebugLogger, logHarbour.LevelDebug2, "spanid9", "", time.Now(), "bhavya", "abcd",
		"newLog", "valueBeingUpdated", "id9", 1, "This is an DEBUG2 logger message 3")

	fmt.Println("---------------------------")
	fmt.Println("---------------------------")
	firstFunc(loggers)
	kafkaConsumer()
}

func firstFunc(loggers logHarbour.LogHandles) {
	logHarbour.LogWrite(loggers.ActivityLogger, logHarbour.LevelInfo, "", "correlationid12", time.Now(), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id12", 1, "Activity logger info Message", logHarbour.DataChg("qty", "1", "2"),
		logHarbour.DataChg("amt", "100", "200"))

	fmt.Println()

	logHarbour.LogWrite(loggers.DataChangeLogger, logHarbour.LevelError, "spanid13", "correlationid13", time.Now(), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id13", 1, "Datachangelogger error Message",
		logHarbour.DataChg("qty", "1", "2"), logHarbour.DataChg("amt", "100", "200"))

	fmt.Println()

	logHarbour.LogWrite(loggers.DebugLogger, logHarbour.LevelDebug1, "spanid14", "correlationid14", time.Now(), "bhavya", "127.0.0.1",
		"newLog", "valueBeingUpdated", "id14", 1, "debug DEBUG1 Message", logHarbour.DataChg("qty", "1", "2"), "key1", "value1")
}

/*
Output:
{"time":"2023-09-27T11:41:27.970812125+05:30","level":"INFO","msg":"This is an activity logger info message","app":"app1","module":"module1","system":"system1","handle":"A","spanId":"spanid1","correlationId":"correlationid1","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id1","status":1,"params":{"somekey":"somevalue","key2":"value2"}}
{"time":"2023-09-27T11:41:27.970870487+05:30","level":"ERROR","msg":"This is an activity logger error message","app":"app1","module":"module1","system":"system1","handle":"A","spanId":"spanid2","correlationId":"correlationid2","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id2","status":1,"params":{"somekey":"somevalue"}}
{"time":"2023-09-27T11:41:27.97087819+05:30","level":"ERROR","msg":"This is an activity logger error message","app":"app1","module":"module1","system":"system1","handle":"A","spanId":"spanid3","correlationId":"correlationid3","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id3","status":1}
----
{"time":"2023-09-27T11:41:27.970886588+05:30","level":"INFO","msg":"This is an DataChange logger info message","app":"app1","module":"module1","system":"system1","handle":"C","spanId":"spanid4","correlationId":"correlationid4","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id4","status":1}
{"time":"2023-09-27T11:41:27.970892532+05:30","level":"INFO","msg":"This is an DataChange logger info message","app":"app1","module":"module1","system":"system1","handle":"C","spanId":"spanid5","correlationId":"correlationid5","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id5","status":1,"params":[{"field":"amt","oldVal":"100","newVal":"200"},{"field":"qty","oldVal":"1","newVal":"2"}]}
{"time":"2023-09-27T11:41:27.970921608+05:30","level":"ERROR","msg":"This is an DataChange logger error message","app":"app1","module":"module1","system":"system1","handle":"C","spanId":"spanid6","correlationId":"correlationid6","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id6","status":1,"params":[{"field":"amt","oldVal":"100","newVal":"200"},{"field":"qty","oldVal":"1","newVal":"2"}]}
----
{"time":"2023-09-27T11:41:27.970938796+05:30","level":"DEBUG0","msg":"This is an DEBUG0 logger message 1","app":"app1","module":"module1","system":"system1","handle":"D","pid":107153,"goVersion":"go1.21.1","source":"sampleMain.go:23","callTrace":"main.main","spanId":"spanid7","correlationId":"correlationid7","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id7","status":1,"params":{"key3":"value3"}}
{"time":"2023-09-27T11:41:27.970956747+05:30","level":"DEBUG1","msg":"This is an DEBUG1 logger message 2","app":"app1","module":"module1","system":"system1","handle":"D","pid":107153,"goVersion":"go1.21.1","source":"sampleMain.go:24","callTrace":"main.main","spanId":"spanid8","correlationId":"correlationid8","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id8","status":1,"params":{"key3":"value3"}}
{"time":"2023-09-27T11:41:27.970967825+05:30","level":"DEBUG2","msg":"This is an DEBUG2 logger message 3","app":"app1","module":"module1","system":"system1","handle":"D","pid":107153,"goVersion":"go1.21.1","source":"sampleMain.go:25","callTrace":"main.main","spanId":"spanid9","correlationId":"correlationid9","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id9","status":1}
---------------------------
---------------------------
{"time":"2023-09-27T11:41:27.97097651+05:30","level":"INFO","msg":"Activity logger info Message","app":"app1","module":"module1","system":"system1","handle":"A","spanId":"spanid12","correlationId":"correlationid12","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id12","status":1}

{"time":"2023-09-27T11:41:27.970983847+05:30","level":"ERROR","msg":"Datachangelogger error Message","app":"app1","module":"module1","system":"system1","handle":"C","spanId":"spanid13","correlationId":"correlationid13","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id13","status":1,"params":[{"field":"qty","oldVal":"1","newVal":"2"},{"field":"amt","oldVal":"100","newVal":"200"}]}

{"time":"2023-09-27T11:41:27.970996614+05:30","level":"DEBUG1","msg":"debug DEBUG1 Message","app":"app1","module":"module1","system":"system1","handle":"D","pid":107153,"goVersion":"go1.21.1","source":"sampleMain.go:37","callTrace":"main.firstFunc","spanId":"spanid14","correlationId":"correlationid14","when":"2023-09-27T11:41:27Z","who":"bhavya","remoteIp":"127.0.0.1","op":"newLog","whatClass":"valueBeingUpdated","whatInstanceId":"id14","status":1,"params":{"key1":"value1"}}
*/
