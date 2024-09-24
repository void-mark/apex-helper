package main

import (
	"database/sql"
	"encoding/base64"
	"flag"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	go_ora "github.com/sijms/go-ora/v2"
)

func dieOnError(msg string, err error) {
	if err != nil {
		fmt.Println(msg, err)
		os.Exit(1)
	}
}

func chunkBy(str *string, chunkSize int) []string {
	var divided []string
	strLen := len(*str)

	for i := 0; i < strLen; i += chunkSize {
		end := i + chunkSize

		if end > strLen {
			end = strLen
		}

		divided = append(divided, (*str)[i:end])
	}

	return divided
}

func makeImportStaticFileScript(appID int, fileName string, source *[]string) string {

	mediaType, _, err := mime.ParseMediaType(mime.TypeByExtension(filepath.Ext(fileName)))

	if err != nil {
		panic("Не удалось определить тип файла")
	}

	script := []string{"declare",
		"  v_application_id apex_applications.application_id%type;",
		"  v_workspace_id   apex_applications.workspace_id%type;",
		"  v_content        clob;",
		"begin",
		"  select application_id, workspace_id",
		"    into v_application_id, v_workspace_id",
		"    from apex_applications",
		fmt.Sprintf("   where application_id = %d;", appID),
		"  apex_util.set_security_group_id (p_security_group_id => v_workspace_id);",
		"  execute immediate 'alter session set current_schema=' || apex_application.g_flow_schema_owner;",
		"  -----------------------------------------------------------------------------------",
		"  dbms_lob.createtemporary(v_content, true, dbms_lob.session);"}

	for _, v := range *source {
		script = append(script, fmt.Sprintf("  dbms_lob.append(v_content, '%s');", v))
	}

	script = append(script, "  wwv_flow_api.create_app_static_file (p_flow_id      => v_application_id,",
		fmt.Sprintf("                                       p_file_name    => '%s',", fileName),
		fmt.Sprintf("                                       p_mime_type    => '%s',", mediaType),
		"                                       p_file_charset => lower('utf-8'),",
		"                                       p_file_content => apex_web_service.clobbase642blob(v_content));",
		"  dbms_lob.freetemporary(v_content);",
		"  commit;",
		"end;")

	return strings.Join(script[:], "\n")
}

func importStatic(conn *sql.DB, appID int, filePath string) {

	source, err := os.ReadFile(filePath)
	dieOnError("Error read file "+filePath+": ", err)

	encodedSource := base64.StdEncoding.EncodeToString(source)
	chunkedSource := chunkBy(&encodedSource, 200)

	script := makeImportStaticFileScript(appID, filepath.Base(filePath), &chunkedSource)

	_, err = conn.Exec(script)

	dieOnError("Error import file "+filePath+": ", err)
	fmt.Println("Successfully import file", filePath)

}

func createConnection(server string, port int, user string, password string, sid string) *sql.DB {
	urlOptions := map[string]string{
		"SID": sid,
	}
	connStr := go_ora.BuildUrl(server, port, "", user, password, urlOptions)
	conn, err := sql.Open("oracle", connStr)
	dieOnError("Error connection to oracle:", err)

	err = conn.Ping()
	if err != nil {
		conn.Close()
	}
	dieOnError("Error ping connection:", err)

	rows, err := conn.Query("SELECT upper(SYS_CONTEXT('USERENV','INSTANCE_NAME')) as n FROM dual")
	dieOnError("Error create query:", err)

	defer rows.Close()

	for rows.Next() {
		var s string
		err := rows.Scan(&s)
		if err != nil {
			break
		}
		fmt.Println("Successfully connected to Oracle instance", s)
	}

	return conn
}

func main() {

	ora_server := flag.String("host", "", "Oracle server IP or URI")
	ora_port := flag.Int("port", 1521, "Oracle server port")
	ora_user := flag.String("user", "", "Oracle user name")
	ora_password := flag.String("password", "", "Oracle user password")
	ora_sid := flag.String("sid", "", "Oracle SID")

	operation := flag.String("op", "", "Operation")
	appID := flag.Int("appID", 0, "Apex application ID")
	file := flag.String("file", "", "Static file path")

	flag.Parse()

	if *operation == "" {
		fmt.Println("Please specify the apex-helper operation using the -operation flag.")
		os.Exit(1)
	}

	conn := createConnection(*ora_server, *ora_port, *ora_user, *ora_password, *ora_sid)
	defer conn.Close()

	switch *operation {
	case "importStatic":
		importStatic(conn, *appID, *file)
	default:
		fmt.Println("Invalid operation. Please specify a valid operation.")
		os.Exit(1)
	}

}
