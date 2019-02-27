# Maître d′

An auth server for [Cube 2: Sauerbraten](http://sauerbraten.org/).


## Why?

This auth server is usable by Sauerbraten game servers like a master server, except that it only understands auth related parts of the protocol, i.e.: `reqauth` and `confauth`. It responds the same way the masterserver at sauerbraten.org would respond to gauth requests (`chalauth`, `succauth`, `failauth`).

It is intended to be used by game servers for a specific auth key domain. The game server decides what privileges, if any, to grant authenticated users of that domain.


## Adding auth keys

[`/cmd/manage/`](/cmd/manage/) contains code for a binary to add users to the database for convenience. It uses an extension of the master server protocol to administrate auth entries. Use it like this:

	export MAITRED_AUTHNAME=pix
	export MAITRED_AUTHKEY=<admin priv key>
	export MAITRED_ADDRESS=localhost:28787

	./manage addauth player1 +daf0f6c8cfbcdf98cc56a316da0702b5423514e98673f1a6
	./manage delauth player1
