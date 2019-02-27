all:
	if [ ! -f maitred.sqlite ]; then sqlite3 maitred.sqlite < maitred.sqlite.schema; fi
	go build ./cmd/maitred
	go build ./cmd/manage

clean:
	rm -f maitred.sqlite ./maitred ./manage