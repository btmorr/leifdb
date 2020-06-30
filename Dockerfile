FROM scratch
ADD build/leifdb /
ADD build/config.toml /.leifdb/config.toml
CMD ["/leifdb", "--data", ".leifdb"]
