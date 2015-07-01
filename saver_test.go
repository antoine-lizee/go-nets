package go_nets

// func TestSaver(t *testing.T) {
// 	filename := "UMtest.xml"
// 	fmt.Println("### TESTING the saver (small file)")
// 	Parser := XmlParser{
// 		FileDir:  "_test/",
// 		FileName: filename,
// 	}
// 	TestSaver := SqlSaver{
// 		dbPath:   "_test/",
// 		dbName:   filename,
// 		DBDriver: "sqlite3",
// 	}
// 	cs := make(chan Filing)
// 	go Parser.Parse(cs, nil)
// 	ListenAndSave(cs, TestSaver)
// }
