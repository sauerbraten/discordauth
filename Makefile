all: bootstrap_db
	go get ./...
	go build ./cmd/maitred
	go build ./cmd/manage

clean:
	rm -f maitred.sqlite ./maitred ./manage

bootstrap_db:
	if [ ! -f maitred.sqlite ];	then \
		sqlite3 maitred.sqlite < maitred.sqlite.schema && \
		sqlite3 maitred.sqlite "insert into users (name, pubkey, admin) values ('pix', '+daf0f6c8cfbcdf98cc56a316da0702b5423514e98673f1a6', 1);"; \
	fi \
