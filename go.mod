module github.com/zafrem/bastion-rag

go 1.26.2

require (
	github.com/zafrem/bastion-sentinel v0.0.0
	github.com/zafrem/bastion-vault v0.0.0
)

replace (
	github.com/zafrem/bastion-sentinel => ./sentinel
	github.com/zafrem/bastion-vault => ./vault
)
