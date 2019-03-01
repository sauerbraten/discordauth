# Maître d′

A master server for [Cube 2: Sauerbraten](http://sauerbraten.org/) that supports centralized stats collection.


## Why?

This master server allows Sauerbraten game servers to authenticate clients with its auth key domain and to report the statistics of authenticated players at the end of a game. It is usable by game servers like the original/vanilla master server, except that it does not keep a server list and does not respond to `list`. The game server decides what privileges, if any, to grant users that it successfully authenticated with this master server.


## Adding auth keys

[`/cmd/manage/`](/cmd/manage/) contains code for a binary to add users to the database for convenience. It uses an extension of the master server protocol to administrate auth entries. Use it like this:

	export MAITRED_AUTHNAME=pix
	export MAITRED_AUTHKEY=<admin priv key>
	export MAITRED_ADDRESS=localhost:28787

	./manage addauth player1 +daf0f6c8cfbcdf98cc56a316da0702b5423514e98673f1a6
	./manage delauth player1
