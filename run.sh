#!/bin/sh

set -a          # -a Each variable or function that is created or modified is given the export
                # attribute and marked for export to the environment of subsequent commands.
                # Ref: https://www.gnu.org/software/bash/manual/html_node/The-Set-Builtin.html

source .env     # Execute the content of the .env file passed as argument, in the current shell
set +a          # Disable the effect of initial "set -a" command


cd cmd && go run *.go