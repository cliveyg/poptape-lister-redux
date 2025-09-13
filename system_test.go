package main

// NewAppForTest replicates main setup but returns *App for use in tests

//func AppTest() *App {
//	err := godotenv.Load()
//	if err != nil {
//		panic("Error loading .env file")
//	}
//
//	var logFile *os.File
//	filePathName := os.Getenv("LOGFILE")
//	logFile, err = os.OpenFile(filePathName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0644)
//	if err != nil {
//		panic(err)
//	}
//
//	// format logline
//	cw := zerolog.ConsoleWriter{Out: logFile, NoColor: true, TimeFormat: time.RFC3339}
//	cw.FormatLevel = func(i interface{}) string {
//		return strings.ToUpper(fmt.Sprintf("[ %-6s]", i))
//	}
//	cw.TimeFormat = "[" + time.RFC3339 + "] - "
//	cw.FormatCaller = func(i interface{}) string {
//		str, _ := i.(string)
//		return fmt.Sprintf("['%s']", str)
//	}
//	cw.PartsOrder = []string{
//		zerolog.LevelFieldName,
//		zerolog.TimestampFieldName,
//		zerolog.MessageFieldName,
//		zerolog.CallerFieldName,
//	}
//
//	logger := zerolog.New(cw).With().Timestamp().Caller().Logger()
//	if os.Getenv("LOGLEVEL") == "debug" {
//		zerolog.SetGlobalLevel(zerolog.DebugLevel)
//	} else if os.Getenv("LOGLEVEL") == "info" {
//		zerolog.SetGlobalLevel(zerolog.InfoLevel)
//	} else {
//		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
//	}
//
//	a := &App{}
//	a.Log = &logger
//	a.InitialiseApp()
//	return a
//}
//
//var a *App
//
//func TestMain(m *testing.M) {
//	a = AppTest()
//	code := m.Run()
//	os.Exit(code)
//}
//
////-----------------------------------------------------------------------------
//// h e l p e r   f u n c t i o n s
////-----------------------------------------------------------------------------
//
//func executeRequest(req *http.Request) *httptest.ResponseRecorder {
//	rr := httptest.NewRecorder()
//	a.Router.ServeHTTP(rr, req)
//	return rr
//}
//
//func checkResponseCode(t *testing.T, expected, actual int) bool {
//	if expected != actual {
//		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
//		return false
//	} else {
//		return true
//	}
//}
//
////-----------------------------------------------------------------------------
//// s t a r t   o f   t e s t s
////-----------------------------------------------------------------------------
//
//func TestAPIStatus(t *testing.T) {
//	req, _ := http.NewRequest("GET", "/list/status", nil)
//	response := executeRequest(req)
//
//	if checkResponseCode(t, http.StatusOK, response.Code) {
//		fmt.Println("[PASS].....TestAPIStatus")
//	}
//}
