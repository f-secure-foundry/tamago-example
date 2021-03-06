module github.com/f-secure-foundry/tamago-example

go 1.16

require (
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/btcsuite/btcutil v1.0.2
	github.com/f-secure-foundry/crucible v0.0.0-20210312172715-ee016f688e57
	github.com/f-secure-foundry/tamago v0.0.0-20210217103808-875e533027e3
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/mkevac/debugcharts v0.0.0-20191222103121-ae1c48aa8615
	github.com/shirou/gopsutil v3.21.1+incompatible // indirect
	golang.org/x/crypto v0.0.0-20201221181555-eec23a3978ad
	golang.org/x/sync v0.0.0-20201207232520-09787c993a3a // indirect
	golang.org/x/sys v0.0.0-20210217090653-ed5674b6da4a // indirect
	golang.org/x/term v0.0.0-20201210144234-2321bbc49cbf
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	gvisor.dev/gvisor v0.0.0-20210213011344-3ef012944d32
)

replace gvisor.dev/gvisor => github.com/f-secure-foundry/gvisor v0.0.0-20210201110150-c18d73317e0f
