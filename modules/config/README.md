# Config Module

*Config module is a part of `Ginject` framework, by default it’s support `.env` only, but you can extends any respective config files by using external libraries.*

- [Config Module](#config-module)
  - [Key Features](#key-features)
  - [Usage](#usage)
  - [`ConfigModuleOptions` Parameters](#configmoduleoptions-parameters)
    - [IsGlobal](#isglobal)
    - [IsIgnoreEnvFile](#isignoreenvfile)
    - [IsOverride](#isoverride)
    - [IsExpandVariables](#isexpandvariables)
    - [ENVFilePaths](#envfilepaths)
    - [OnInit](#oninit)
    - [Loads (Custom Configuration Files)](#loads-custom-configuration-files)
    - [Hooks](#hooks)
  - [`ConfigService` Methods](#configservice-methods)
    - [Get](#get)
      - [Parameters](#parameters)
      - [Returns](#returns)
      - [Usage](#usage-1)

## Key Features
- Zero-dependency
- Multiline values
- Variable expansion
- Extend for various file types
- Access variable via namespace

## Usage

First you need create a `.env` file in the root of your project:

```dosini
USERNAME=ginject # Support comments
PASSWORD=123$%^
```

Import config module into main module, by default config module will read `.env` in the root of your project.
Create a provider and inject `ConfigService`.
Let's take a look in an example following `main.go`:

```go
package main

import (
	"fmt"

	"github.com/dangduoc08/ginject/core"
	"github.com/dangduoc08/ginject/modules/config"
)

type DBProvider struct {
	ConfigService config.ConfigService
}

func (dbProvider DBProvider) Inject() core.Provider {
	fmt.Println("USERNAME:", dbProvider.ConfigService.Get(("USERNAME")))
	fmt.Println("PASSWORD:", dbProvider.ConfigService.Get(("PASSWORD")))
	return dbProvider
}

func main() {
	app := core.New()

	app.Create(
		core.ModuleBuilder().
			Imports(
				config.Register(&config.ConfigModuleOptions{}),
			).
			Providers(DBProvider{}).
			Build(),
	)
}
```

Finally, run:
```shell
go run main.go
```

There are logs print out on console:
```console
USERNAME: ginject
PASSWORD: 123$%^
```

## `ConfigModuleOptions` Parameters

### IsGlobal
Type: `bool`

Default: `false`

Required: `false`

Set module as globally. By set to `true`, you don't need to import config module wherever you read env variables.

```go
config.Register(&config.ConfigModuleOptions{
  IsGlobal: true,
})
```

### IsIgnoreEnvFile
Type: `bool`

Default: `false`

Required: `false`

If you don't want to use `.env` file, instead would like to simply access `OS` environment variables from the runtime environment. set `IsIgnoreEnvFile: true`

```go
config.Register(&config.ConfigModuleOptions{
  IsIgnoreEnvFile: true,
})
```

### IsOverride
Type: `bool`

Default: `false`

Required: `false`

Override any environment variables that have already been set on your machine with values from your .env file.

```go
config.Register(&config.ConfigModuleOptions{
  IsOverride: true,
})
```

### IsExpandVariables
Type: `bool`

Default: `false`

Required: `false`

Module support for environment variable expansion. With this technique, you can create nested environment variables, where one variable is referred to within the definition of another, see the following example:

```dosini
HOST=localhost:27017
USER=admin
PWD=secret
DB=test
URI=mongodb://${USER}:${PWD}@${HOST}/${DB}
```

```go
config.Register(&config.ConfigModuleOptions{
  IsExpandVariables: true,
})
```

```go
fmt.Println("URI:", dbProvider.ConfigService.Get(("URI")))
```

Console:
```console
URI: mongodb://admin:secret@localhost:27017/test
```

### ENVFilePaths
Type: `[]string`

Default: `[]string{".env"}`

Required: `false`

By default, the module looks for a .env file in the root directory of your project. To specify another paths for the .env files, set the `ENVFilePaths` property to list of files, as follows:

```go
config.Register(&config.ConfigModuleOptions{
  ENVFilePaths: []string{".env.sql", ".env.aws", "env.redis"},
})
```

### OnInit
Type: `func()`

Default: `nil`

Required: `false`

`OnInit` is a module's life cycle which is invoked before Config module was injected.

### Loads (Custom Configuration Files)
Type: `[]func() map[string]interface{}`

Default: `[]func() map[string]interface{}{}`

Required: `false`

For complex projects, sometimes `.env` is not enough, you want to various file types in their project or simply you prefer using another files types. `Loads` is property that allows you pass load function into array. By this approach, you can easy using any files you want, as long as function return `map[string]interface{}` type.

Let's take a look in 2 below examples:

Firstly, create `configuration` function:
```go
func configuration() map[string]interface{} {
	return map[string]interface{}{
		"S3_BUCKET":  "This is your S3 bucket",
		"SECRET_KEY": "This is you secret key",
	}
}
```

Then, add into `Loads` property
```go
config.Register(&config.ConfigModuleOptions{
  Loads: []config.ConfigLoadFn{configuration},
})
```

Print `S3_BUCKET` and `SECRET_KEY` on console:
```console
This is your S3 bucket
This is you secret key
```

Now, let's see how can we read another file types. Here is an example of a configuration using `YAML` format:

We need install go package to encode and decode YAML values. For this example, we will use `yaml.v3`.

Install:
```shell
go get gopkg.in/yaml.v3
```

Create an sample `config.yaml`:
```yaml
database:
  host: localhost
  port: 5432
  name: mydatabase
  username: myuser
  password: mypassword

server:
  host: localhost
  port: 8080
  debug: true

logging:
  level: info
  file_path: /var/log/myapp.log
```

Edit `configuration` function:
```go
import (
	"os"

	"gopkg.in/yaml.v3"
)

func configuration() map[string]interface{} {
	b, err := os.ReadFile("config.yaml")
	if err != nil {
		panic(err)
	}

	config := make(map[string]interface{})
	err = yaml.Unmarshal(b, &config)
	if err != nil {
		panic(err)
	}

	return config
}
```

You can accress variables via namespace:
```go
provider.ConfigService.Get(("database"))
// Will print: map[host:localhost name:mydatabase password:mypassword port:5432 username:myuser]
provider.ConfigService.Get(("database.port"))
// Will print: 5432
```

### Hooks
Type: `[]func(ConfigService)`

Default: `[]func(ConfigService)`

Required: `false`

Hooks is a property that allow you catch env map before config module instance is created. The idea of hook is give you ability to transforms or validations configuration to avoid mismatch configs before application start.

Hook functions will be invoked sequency in array.

```go
config.Register(&config.ConfigModuleOptions{
  Hooks: []config.ConfigHookFn{
    func(c config.ConfigService) {
      c.Set("ANOMYNOUS") = "John Doe"
    },
  },
}),
```
```go
provider.ConfigService.Get(("ANOMYNOUS")) // Will print: John Doe
```

It is standard practice to throw an exception during application startup if required environment variables haven't been provided or if they don't meet certain validation rules.

In this example we will use `validator` package to validate struct tags. First, we need install library.

Install:
```shell
go get github.com/go-playground/validator/v10
```

Add env key like below sample:

```dosini
EMAIL=duocdevtest@gmail
```

Define a struct:

```go
type Config struct {
	EMAIL string `validate:"email"`
}
```

Finally, on `Hooks` property add function:

```go
config.Register(&config.ConfigModuleOptions{
  Hooks: []config.ConfigHookFn{
    func(c config.ConfigService) {
      errs := validator.New().Struct(Config{
        EMAIL: c.Get("EMAIL").(string),
      })

      if errs != nil {
        panic(errs)
      }
    },
  },
}),
```

This will throw exception:
```shell
panic: Key: 'Config.EMAIL' Error:Field validation for 'EMAIL' failed on the 'email' tag
```

## `ConfigService` Methods

### Get

Method help to get env value by key. Return `nil` if hasn't value.

#### Parameters
- 1st parameter: `string`

- Description: Env key name


#### Returns
- 1st value: `interface{}`

- Description: Env key value

#### Usage

```go
var PORT int = provider.ConfigService.Get(("PORT")).(int)
```
