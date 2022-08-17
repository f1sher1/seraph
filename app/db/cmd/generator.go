package main

import (
	"seraph/app/db"
	"seraph/app/db/models"

	"gorm.io/gen"
)

// generate code
func main() {
	// specify the output directory (default: "./query")
	// ### if you want to query without context constrain, set mode gen.WithoutContext ###
	g := gen.NewGenerator(gen.Config{
		OutPath: "app/db/query",

		ModelPkgPath: "app/db/models",

		/* Mode: gen.WithoutContext|gen.WithDefaultQuery*/
		//if you want the nullable field generation property to be pointer type, set FieldNullable true
		/* FieldNullable: true,*/
		//if you want to generate index tags from database, set FieldWithIndexTag true
		FieldWithIndexTag: true,
		//if you want to generate type tags from database, set FieldWithTypeTag true
		/* FieldWithTypeTag: true,*/
		//if you need unit tests for query code, set WithUnitTest true
		/* WithUnitTest: true, */
	})

	// reuse the database connection in Project or create a connection here
	// if you want to use GenerateModel/GenerateModelAs, UseDB is necessary or it will panic
	// db, _ := gorm.Open(mysql.Open("root:@(0.0.0.0:3306)/demo?charset=utf8mb4&parseTime=True&loc=Local"))
	//conn := "mysql://opstk:123456@0.0.0.0:3306/seraph?charset=utf8&parseTime=True&loc=Local"
	cfg := &db.Config{
		Connection:  "mysql://root:123456@0.0.0.0:3306/seraph?charset=utf8&parseTime=True&loc=Local",
		Debug:       true,
		PoolSize:    0,
		IdleTimeout: 0,
	}

	if err := db.Init(cfg); err != nil {
		panic(err.Error())
	}
	g.UseDB(db.GetDBConnection())

	// apply basic crud api on structs or table models which is specified by table name with function
	// GenerateModel/GenerateModelAs. And generator will generate table models' code when calling Execute.
	//g.ApplyBasic(model.User{}, g.GenerateModel("company"), g.GenerateModelAs("people", "Person", gen.FieldIgnore("address")))
	//
	//// apply diy interfaces on structs or table models
	//g.ApplyInterface(func(method model.Method) {}, model.User{}, g.GenerateModel("company"))

	//g.GenerateModelAs("workflow_execution_v2", "WorkflowExecutionV2")
	//g.ApplyBasic(g.GenerateAllTable()...)
	//g.ApplyBasic(g.GenerateModelAs("workflow_execution_v2", "WorkflowExecutionV2"))

	g.ApplyBasic(models.Models...)

	// execute the action of code generation
	g.Execute()
}
