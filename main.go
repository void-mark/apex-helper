package main

import "github.com/void-mark/apex-helper/helper"

func main() {

	app := helper.NewHelper()
	err := app.Start()
	if err == nil {
		defer app.Close()

		err = app.ExecuteOperation()
	}

	app.DieOnError("apex-helper:", err)

}
