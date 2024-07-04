# ns1_ddns
NS1 Dynamic DNS Updater

Build using `docker build . -t rhm176/ns1_ddns`

Environment variables:
* `NS1_KEY` Required, API Key from NS1
* `NS1_ZONE` Required, NS1 Zone name
* `NS1_DOMAINS` NS1 Domain names to update, seperated by a comma. Currently only A (IPv4) is supported. Defaults to `NS1_ZONE`
* `NS1_INTERVAL` Interval to check if IPv4 changed. Defaults to `10`
