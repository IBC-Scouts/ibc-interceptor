# Quick and dirty (dont judge me!) script to simulate common local flow.
import os
import sys

buildDir = "build"
binName = "interceptor"

# Add argument for input path to optimism folder
def GetOPDir():
    if len(sys.argv) != 2:
        print("Error: Invalid number of arguments, pass in relative path for local OP repository")
        sys.exit(1)

    print(f"Running e2e tests for {sys.argv[1]}")

    return sys.argv[1]


if __name__ == "__main__":
    optimism_path = GetOPDir()
    optimism_devnet_path = os.path.join(optimism_path, '.devnet')
    optimism_e2e_path = os.path.join(optimism_path, 'op-e2e')

    # assert directory named 'optimism' exists in parent directory
    if not os.path.exists(optimism_path):
        print("Error: 'optimism' directory not found in parent directory")
        sys.exit(1)

    # assert it has a `.devnet` folder.
    if not os.path.exists(optimism_devnet_path):
        print("Error: '.devnet' directory not found in 'optimism' directory")
        sys.exit(1)

    # if we have any running interceptor processes, kill them

    # kill any running interceptor processes
    os.system(f'killall -e -9 {binName}')

    # build the interceptor
    os.system('make build-interceptor')
    # copy to optimism/op-e2e
    os.system(f'cp {buildDir}/{binName} {optimism_e2e_path}')

    # run just the Deposit test for now.
    os.chdir(optimism_e2e_path)
    os.system('go clean -testcache && go test -v -run TestDepositTxCreateContract ./...')