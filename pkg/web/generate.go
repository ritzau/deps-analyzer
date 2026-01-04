package web

//go:generate sh -c "test -d static/src && $(go env GOPATH)/bin/esbuild static/src/app.ts --bundle --outfile=static/app.js --target=es2020 --sourcemap || echo 'Skipping TypeScript compilation (no static/src directory)'"

