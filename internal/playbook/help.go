package playbook

import "fmt"

// PrintHelp prints playbook-specific usage guidance.
func PrintHelp(name string) {
	switch name {
	case "dmr":
		fmt.Println("Docker Model Runner (dmr) playbook")
		fmt.Println("Commands:")
		fmt.Println("  setup       - Install Docker + GPU runtime prerequisites on the DGX")
		fmt.Println("  install     - Install/upgrade the Docker Model Runner controller")
		fmt.Println("  update      - Reinstall the controller with fresh bits")
		fmt.Println("  status      - Check Docker Model Runner status")
		fmt.Println("  logs        - Tail controller logs (pass extra args like --tail 100)")
		fmt.Println("  list        - List cached models (same as 'docker model list')")
		fmt.Println("  pull        - Pull models from Docker Hub/HF/nvcr.io (usage: dgx run dmr pull <ref>)")
		fmt.Println("  run         - Run a model with a single prompt (usage: dgx run dmr run <ref> \"prompt\")")
		fmt.Println("  uninstall   - Remove the controller and cached images")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  dgx run dmr setup")
		fmt.Println("  dgx run dmr install")
		fmt.Println("  dgx run dmr pull ai/smollm2:360M-Q4_K_M")
		fmt.Println("  dgx run dmr run ai/smollm2:360M-Q4_K_M \"Explain quantum computing\"")
		fmt.Println("  dgx run dmr status")
		fmt.Println("  dgx run dmr logs --tail 100")
	default:
		fmt.Printf("No dedicated help available for playbook '%s'. Refer to README/PLAYBOOKS for usage.\n", name)
	}
}
