package helper

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"mime"
	"os"
	"path/filepath"

	go_ora "github.com/sijms/go-ora/v2"
	"github.com/void-mark/apex-helper/utils"
)

// структура параметров командной строки приложения
type cliArg struct {
	oraServer   *string
	oraPort     *int
	oraUser     *string
	oraPassword *string
	oraSid      *string
	operation   *string
	appId       *int
	file        *string
}

// структура приложения
type helper struct {
	args     *cliArg
	db       *sql.DB
	notifier Notifier
}

// конструктор приложения командной строки
func NewHelper() *helper {
	return &helper{
		&cliArg{
			oraServer:   flag.String("host", "", "Oracle server IP or URI"),
			oraPort:     flag.Int("port", 1521, "Oracle server port"),
			oraUser:     flag.String("user", "", "Oracle user name"),
			oraPassword: flag.String("password", "", "Oracle user password"),
			oraSid:      flag.String("sid", "", "Oracle SID"),
			operation:   flag.String("operation", "", "Operation"),
			appId:       flag.Int("appId", 0, "Apex application ID"),
			file:        flag.String("file", "", "Static file path"),
		},
		nil,
		NewColorNotifier(),
	}
}

// метод начала работы приложения
func (a *helper) Start() error {
	flag.Parse()

	if *a.args.operation == "" {
		return errors.New("отсутствует тип выполняемой операции, укажите тип операции использую флаг -operation")
	}

	return nil
}

// метод завершения приложения
func (a *helper) Close() {
	if a.db != nil {
		a.db.Close()
	}
}

// выполнение указанной операции
func (app *helper) ExecuteOperation() error {
	var err error

	err = app.createConnection()

	if err == nil {
		switch *app.args.operation {
		case "importStatic":
			err = app.importStatic()
		default:
			err = errors.New(fmt.Sprintf("указанная операция \"%s\" не имеет обработчика", *app.args.operation))
		}
	}

	return err
}

// вспомогательный метод который выводит текст ошибки и завершает приложение с кодом 1
func (app *helper) DieOnError(msg string, err error) {
	if err != nil {
		app.notifier.bad(fmt.Sprint(msg, err))
		os.Exit(1)
	}
}

// импортирование файла в APEX приложение
func (app *helper) importStatic() error {

	importErr := errors.New("ошибка импорта файла")

	fileName := filepath.Base(*app.args.file)
	mediaType, _, err := mime.ParseMediaType(mime.TypeByExtension(filepath.Ext(fileName)))

	if err != nil {
		return errors.Join(importErr, err)
	}

	source, err := os.ReadFile(*app.args.file)
	if err != nil {
		return errors.Join(importErr, err)
	}

	encodedSource := base64.StdEncoding.EncodeToString(source)
	chunkedSource := utils.ChunkBy(&encodedSource, 200)

	script, err := utils.MakeImportStaticFileScript(*app.args.appId, fileName, mediaType, &chunkedSource)
	if err != nil {
		return errors.Join(importErr, err)
	}

	_, err = app.db.Exec(script)
	if err != nil {
		return errors.Join(importErr, err)
	}

	app.notifier.success(fmt.Sprintf("Файл %s успешно импортирован в приложение %d", *app.args.file, *app.args.appId))

	return nil

}

func (app *helper) createConnection() error {
	var err error

	urlOptions := map[string]string{
		"SID": *app.args.oraSid,
	}
	connStr := go_ora.BuildUrl(*app.args.oraServer, *app.args.oraPort, "", *app.args.oraUser, *app.args.oraPassword, urlOptions)
	app.db, err = sql.Open("oracle", connStr)
	if err != nil {
		return errors.Join(errors.New("ошибка подключения к БД"), err)
	}

	err = app.db.Ping()
	if err != nil {
		app.db.Close()
		return errors.Join(errors.New("ошибка проверки связи с БД"), err)
	}

	rows, err := app.db.Query("SELECT upper(SYS_CONTEXT('USERENV','INSTANCE_NAME')) as n FROM dual")
	if err != nil {
		return errors.Join(errors.New("ошибка выполнения запроса"), err)
	}

	defer rows.Close()

	for rows.Next() {
		var s string
		if err = rows.Scan(&s); err != nil {
			break
		}
		app.notifier.success(fmt.Sprintf("Успешно выполнено подключение к экземпляру %s БД Oracle", s))
	}

	return nil
}
