# websocket multiplexing
## 功能说明
将多路客户端连接合并为单路连接的websocket反向代理服务器。


允许将来自客户端的多个websocket连接托管到当前应用上，当源服务器推送数据过来后，
当前服务会将数据分别发送个每个订阅该数据源的客户端连接