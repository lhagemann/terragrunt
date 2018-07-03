# errors from sending output to Pipe in RunShell Command


An example of the start of the run for a passing test, where the output is simply being captured in `RunShellCommandAndCaptureOutput`

```
$ go test -v -run TestInitRetry -parallel 128                                                                                                                             (transient_errors✱)
=== RUN   TestInitRetry
=== PAUSE TestInitRetry
=== CONT  TestInitRetry
[terragrunt] [/var/folders/sl/vp01y4852110fgxqqlpzdy340000gn/T/terragrunt-test585895894/fixture-retryableerrors/retry-init] 2018/07/02 20:11:25 Running command: terraform --version
[terragrunt] 2018/07/02 20:11:25 Reading Terragrunt config file at /var/folders/sl/vp01y4852110fgxqqlpzdy340000gn/T/terragrunt-test585895894/fixture-retryableerrors/retry-init/terraform.tfvars
[terragrunt] [/var/folders/sl/vp01y4852110fgxqqlpzdy340000gn/T/terragrunt-test585895894/fixture-retryableerrors/retry-init] 2018/07/02 20:11:25 Running command: terraform init
[terragrunt] [/var/folders/sl/vp01y4852110fgxqqlpzdy340000gn/T/terragrunt-test585895894/fixture-retryableerrors/retry-init] 2018/07/02 20:11:27
Initializing provider plugins...
- Checking for available provider plugins on https://releases.hashicorp.com...
- Downloading plugin for provider "null" (1.0.0)...

The following providers do not have any version constraints in configuration,
so the latest version was installed.
```


But, when `RunShellComand` is modified to use the `cmd.Std*Pipe()` we fail early in the process. Note the output of `terraform --version` is there now, 

```
[lisa@chakram88:...runtwork-io/terragrunt/test]$ go test -v -run TestInitRetry -parallel 128                                                                                                                                  (show_output✱)
=== RUN   TestInitRetry
=== PAUSE TestInitRetry
=== CONT  TestInitRetry
[terragrunt] 2018/07/02 20:46:05 Running command: terraform --version
[terragrunt] [/var/folders/sl/vp01y4852110fgxqqlpzdy340000gn/T/terragrunt-test523147508/fixture-retryableerrors/retry-init] 2018/07/02 20:46:05 Running Shell command: terraform --version
[terragrunt] [/var/folders/sl/vp01y4852110fgxqqlpzdy340000gn/T/terragrunt-test523147508/fixture-retryableerrors/retry-init] 2018/07/02 20:46:05 Terraform v0.11.7
[terragrunt] [/var/folders/sl/vp01y4852110fgxqqlpzdy340000gn/T/terragrunt-test523147508/fixture-retryableerrors/retry-init] 2018/07/02 20:46:05
--- FAIL: TestInitRetry (0.34s)
	integration_test.go:842: Copying fixture-retryableerrors/retry-init to /var/folders/sl/vp01y4852110fgxqqlpzdy340000gn/T/terragrunt-test523147508
	integration_test.go:791:
			Error Trace:	integration_test.go:791
			Error:      	Expected nil, but got: &errors.Error{Err:"", stack:[]uintptr{0x155a80b, 0x155a515, 0x1554d81, 0x14f7f98, 0x14f60bc, 0x1565a12, 0x1565311, 0x10f15e0, 0x105ba51}, frames:[]errors.StackFrame(nil), prefix:""}
			Test:       	TestInitRetry
	integration_test.go:792:
	integration_test.go:793:
			Error Trace:	integration_test.go:793
			Error:      	"" does not contain "may need to run terraform init"
			Test:       	TestInitRetry
FAIL
exit status 1
FAIL	github.com/gruntwork-io/terragrunt/test	0.358s
```