# 一、AddDoc

1. 对Doc进行哈希，并对哈希值取模，选出对应的Group
2. 遍历Group中的每个节点（endpoint），存入切片中
3. 遍历切片，对每个元素（endpoint）建立gRPC连接
4. 对每个gRPC连接发送AddDoc请求
5. 返回成功添加Doc节点的个数

# 二、DeleteDoc

1. 对Doc进行哈希，并对哈希值取模，选出对应的Group
2. 遍历Group中的每个节点（endpoint），存入切片中
3. 遍历切片，对每个元素（endpoint）建立gRPC连接
4. 对每个gRPC连接发送DeleteDoc请求
5. 返回成功删除Doc节点的个数

# 三、SearchDoc

1. 对Doc进行哈希，并对哈希值取模，选出对应的Group
2. 遍历Group中的每个节点（endpoint），存入切片中
3. 使用负载均衡算法，选择一个节点进行搜索
4. 返回搜索到的文档
