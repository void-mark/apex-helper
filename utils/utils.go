package utils

import (
	"fmt"
	"strings"
)

func ChunkBy(str *string, chunkSize int) []string {
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

func MakeImportStaticFileScript(appID int, fileName string, mediaType string, source *[]string) (string, error) {

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
		"  dbms_lob.createTemporary(v_content, true, dbms_lob.session);"}

	for _, v := range *source {
		script = append(script, fmt.Sprintf("  dbms_lob.append(v_content, '%s');", v))
	}

	script = append(script, "  wwv_flow_api.create_app_static_file (p_flow_id      => v_application_id,",
		fmt.Sprintf("                                       p_file_name    => '%s',", fileName),
		fmt.Sprintf("                                       p_mime_type    => '%s',", mediaType),
		"                                       p_file_charset => lower('utf-8'),",
		"                                       p_file_content => apex_web_service.clobBase642blob(v_content));",
		"  dbms_lob.freeTemporary(v_content);",
		"  commit;",
		"end;")

	return strings.Join(script[:], "\n"), nil
}
