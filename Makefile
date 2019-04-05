.PHONY: all dependencies maitred manage stats clean rebuild_db

all: dependencies maitred manage stats

dependencies:
	go get ./...

maitred:
	go build ./cmd/maitred

manage:
	go build ./cmd/manage

stats:
	go build ./cmd/stats

clean:
	rm -f ./maitred ./manage ./stats


# utilities

rebuild_db:
	if [ ! -f maitred.sqlite ];	then \
		for m in migrations/*.up.sql; do sqlite3 maitred.sqlite < "$$m"; done \
	fi \