package cli

// run runs a command in the container without mounting the user environment.
//
// Workflow
//  - Build applicationin a

// Source Code example
// helloworld/
//   cneproject
//   cneignore
//   .git
//   .gitignore
//   README.md
//   sources
//   build

// Script example, e.g. python
// llm/
//   cneproject
//   cneignore
//   .git
//   .gitignore
//   README.md
//   register.txt???
//
//

// needs a
//   - default user and group
//

func runRunE(cmd *cobra.Command, args []string) error {

	pending, err := imageStatus(prj)
	if err != nil {
		return err
	}
	if pending {
		return nil // some error?? InUse? or InvalidArgument?
	}

	return nil
}
