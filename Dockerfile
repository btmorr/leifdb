FROM scratch
ADD build/leifdb /
CMD ["/leifdb"]

# Build:
#   docker build -t <tag>:<version> .
# Example run, with 3 containers at IP addresses <container_1_ip>, <container_2_ip>, and <container_3_ip>:
#   docker run -it <tag>:<version> \
#     -e LEIFDB_MEMBER_NODES="<container_1_ip>:16990,<container_2_ip>:16990,<container_3_ip>:16990" \
#     -p 8080:8080 -p 16990:16990
