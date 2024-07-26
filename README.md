# 优化SQLite让其适用于一般服务或大型服务

参考资料：https://mp.weixin.qq.com/s/ZweoPdIda7mVi1bDW-90cg

# 核心概要

```sql
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = 1000000000;
PRAGMA foreign_keys = true;
PRAGMA temp_store = memory;
使用BEGIN IMMEDIATE事务；
writeDB.SetMaxOpenConns(1)；
readDB.SetMaxOpenConns(max(4, runtime.NumCPU()))；
使用STRICT表。
```

---
created: 2024-07-25T10:15:59 (UTC +08:00)
tags: []
source: https://mp.weixin.qq.com/s/ZweoPdIda7mVi1bDW-90cg
author: 虫虫搜奇
---

# 优化SQLite，让其适用于一般服务或大型服务

> ## Excerpt
> SQLite经常被误解为“玩具数据库”，或者只是“现代版的Access，一个文件数据库而已”，只适用于“迷你小

---
SQLite经常被误解为“玩具数据库”，或者只是“现代版的Access，一个文件数据库而已”，只适用于“迷你小网站的数据库”。然而事实是，SQLite到处都在用：大量手机APP和嵌入式系统都在使用。有些有点了解、初步尝试过的人也会常常遇到“SQLITE_BUSY”问题，而觉得SQLite不行，不能扛大梁。实际上是因为，其是因为SQLite默认配置针对嵌入式用例进行了优化，对于其他应用需要一些设置才可以发挥出其无限的潜能，这儿虫虫就教大家如何优化SQLite使其能胜任通用的数据库任务，甚至是大型数据库的工作。

![图片](https://mmbiz.qpic.cn/sz_mmbiz_png/hMIkz6ibKC9tkdsJUic78ugWKMINXyntoHG7rFXmRl7pOzyOjVN1rHILNldj2jvnHhMdbe0FuKqicFN8d27hKAGBQ/640?wx_fmt=png&from=appmsg&tp=webp&wxfrom=5&wx_lazy=1&wx_co=1)

## 概述

SQLite是当今部署和使用最广泛的数据库引擎。SQLite几乎无处不在，一部手机和计算机上可能有数百个SQLite数据库（取决于应用程序的数量），其他智能汽车中也在用，甚至的飞机上火箭上都在用。这些实际应用都证明了SQLite可靠性和安全性。

## 为何选择SQLite

选择SQLite用于服务最主要原因是可靠性、性能和简单性。
**可靠性**：当消除对数据库的网络调用时，就消除了大量的故障情况，例如DNS错误和网络中断。
**性能**：网络数据库的读取延迟为毫秒级，而SQLite的读取延迟为微秒级。
**简单性**：性能和可靠性带来简单性。使用SQLite可以在单台机器上疯狂地扩展，并消除围绕集群管理（比如k8s和co）的所有麻烦。更好的是，它大大提高了安全性：不再有数据库暴露在公众面前，也不再有传输过程中配置错误的加密...这对于私有和本地项目尤其重要，在这些项目中，用户可能希望在不成为系统管理员的情况下自行托管应用程序安全工程师

## SQLITE_BUSY错误

SQLITE_BUSY（或database is locked，取决于使用的编程语言）是在开始使用SQLite时最常遇到的错误，因此探究其意义及其发生原因非常重要。

默认情况下SQLite只允许1个写入者同时写入数据库。为此，SQLite引擎会锁定数据库以进行写入。

因此，SQLITE_BUSY的意思很简单：当尝试获取数据库的锁，但失败了，因为单独的连接/进程持有冲突的锁”。

如果不同的线程/进程尝试同时写入数据库，SQLite将稍等片刻并重试。busy_timeout重试毫秒数后，SQLite返回SQLITE_BUSY错误。

## **优化SQLite**

默认情况下，SQLite针对客户端应用程序（例如移动应用程序和嵌入式设备）进行了优化，但通过正确的设置，也可以使其非常适合服务器。

### 配置PRAGMA

PRAGMA命令，使用简单的应在打开连接后立即向数据库发出db.Exec调用或类似的指令。

```sql
PRAGMA journal_mode = WAL;
```

WAL日志模式提供预写日志，提供更多并发性，因为读取器不会阻止写入器，写入器也不会阻止读取器，这与读取器阻止写入器的默认模式相反，反之亦然。

```sql
PRAGMA synchronous = NORMAL;
```

Synchronous配置为NORMAL时，SQLite 数据库引擎将在最关键的时刻进行同步，但频率低于FULL模式。WAL模式在synchronous=NORMAL时可以避免损坏。

它通过WAL模式提供最佳性能。

```sql
PRAGMA cache_size = 1000000000;
```

### 增加SQLite缓存

当使用cache_sizepragma命令更改缓存大小时，更改仅在当前会话中持续。 当数据库关闭并重新打开时，缓存大小将恢复为默认值。

```sql
PRAGMA foreign_keys = true;
```

默认情况下，由于历史原因，SQLite不强制执行外键，需要手动启用它们。

```sql
PRAGMA busy_timeout = 5000;
```

正如之前看到的，设置一个更大的busy_timeout有助于防止SQLITE_BUSY错误。

对于面向用户的应用程序（例如API），建议使用5000（5秒）；后端应用程序（例如队列或应用模块间调用的API），可设置为15000（15秒）或更大。

### 使用即时事务

默认情况下，SQLite在DEFERRED模式启动事务，它们被视为只读。当发出包含写入/更新/删除语句的查询时，它们会升级为需要动态数据库锁定的写入事务。 问题是，通过在事务启动后升级事务，SQLite将立即返回一个SQLITE_BUSY错误而不是检查busy_timeout设置。如果数据库已被另一个连接锁定。

这就是为什么应该在BEGIN IMMEDIATE开始事务，而非仅仅BEGIN。如果事务开始时数据库被锁定，SQLite会检查busy_timeout设置。

这个配置，在Python的Django 5.1+中可以通过：

```sql
"transaction_mode": "IMMEDIATE"
```

在Golang应用中，可得数据池配置中通过

```go
mydb.db?_txlock=immediate
```

### 1个写连接，多个读连接

最后，要彻底消除SQLITE_BUSY错误的方法是使用1个写连接进行写入和查询连接并将其保护在互斥锁后面。

在Golang中可以通过设置

```go
db.SetMaxOpenConns(1)
```

因此，写锁将由互斥体管理，写操作在应用程序端排队，而不是依赖于SQLite内置的重试和busy_timeout机制。

当然这样读取性能将会很差，因为读取查询将与单个数据库连接的写入查询竞争，但正如我们所看到的， WAL在日志模式下，即使ddatabase被锁定以进行写入，SQLite也允许无限读取。

诀窍是设置2个数据库连接池：1个用于写入的池，1个读取池，其根据CPU数量进行扩展。

```go
writeDB, err := sql.Open("sqlite3", connectionUrl)
if err != nil {
// ...
}
writeDB.SetMaxOpenConns(1)

readDB, err := sql.Open("sqlite3", connectionUrl)
if err != nil {
// ...
}
readDB.SetMaxOpenConns(max(4, runtime.NumCPU()))
```

具体代码实现时候，可以在需要调Select/Get调用时使用ReadDB连接池，而要进行Exec/ExecSelect/Transaction方法使用WriteDB连接池。

```go
type DB struct {
writeDB *sqlx.DB
readDB *sqlx.DB
}
func (db *DB) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
return db.writeDB.ExecContext(ctx, query, args...)
}
func (db *DB) Select(ctx context.Context, dest any, query string, args ...any) error {
return db.readDB.SelectContext(ctx, dest, query, args...)
}
// ...
```

### 在单个事务中批量写入

如果应用程序允许，可以在单个事务中进行批量写入：

```go
err = Db.Transaction(func (tx *Tx) (err error) {
for _, event := range events {
err = tx.Exec("...", ...)
if err != nil {
return err
}
return nil
}
})
```

在批量插入时，SQLite在上述一般商用服务器上可以轻松达到每秒约25W+写入。

### 分片

当硬件写入性能达到SQLite的限制时，可将数据库根据业务模块拆分为多个较小的部分（读写分离）。

例如，一个典型的应用程序可使用3个数据库：

service.db这是包含所有用户和业务数据的主数据库：75%+读取/25%写入。

service_queue.db这是用于后台作业的队列：98%+写入/2%读取，主要是UPDATE操作。

service_events.db是存储所有事件的地方。 90%写入/10%读取。

事件数据库是写入最密集的数据库，可设置20毫秒或每次内存中事件缓冲区达到10,000条记录时，在单个事务中批量插入事件。

### 使用STRICT表

默认情况下，SQLite 是“弱类型”，可以将字符串插入到INT字段，这很多时候会造成程序bub并严重影响性能。 =

从3.37开始，SQLite支持 STRICT表的模式并将强制强类型化。

```sql
CREATE TABLE cc (
id BLOB NOT NULL PRIMARY KEY,
created_at INTEGER NOT NULL,
something INT NOT NULL
) STRICT;
```

## 操作和备份

### Litestream

Litestream 是SQLite秘密武器和。Litestream 提供“SQLite 的流式复制”。

Litestream是一个守护进程，它在后台复制数据库的WAL并将其存储在S3等兼容服务器上。它为SQLite数据库启用（异步）时间点恢复和实时备份，并且在发生中断时，可以无损地恢复数据库。 所有这一切只需要每月几美分的S3成本。

![图片](https://mmbiz.qpic.cn/sz_mmbiz_png/hMIkz6ibKC9tkdsJUic78ugWKMINXyntoHBibTXusVhO6N7gIAkVGtmsZtgFrr5aNY9rI3hPxGqX6QxlodYw9icbfw/640?wx_fmt=png&from=appmsg&tp=webp&wxfrom=5&wx_lazy=1&wx_co=1)

### 文件系统

SQLite官方建议不在网络文件系统上使用SQLite，因为某些网络文件系统具有错误的锁定机制，如果多个进程访问数据库，可能会导致数据库损坏。

但这种情况只有在多个进程/机器同时访问多机网络文件系统（NFS）时才会发生。 在具有单机文件系统但连接到网络卷（例如AWS EBS或Scaleway Block Storage）的计算机上，内核负责数据库的性能和正确锁定。

可以两个选择。

在裸机服务器上使用SQLite，并在RAID配置中使用多个磁盘来提高可靠性和可用性。

或者，如果在云中使用SQLite，建议将SQLite数据库放在只能由一台机器访问的网络卷上的单机文件系统(ext4)上，例如AWS EBS或Scaleway Block Storage，并具有足够的IOPS。这些网络卷比本地SSD提供更高的可靠性和可用性。

### 零停机应用程序更新

使用SO_REUSEPORT套接字选项，这样可以：

在与旧版本相同的端口上启动新版本的应用程序

停止旧的应用程序

### 服务器零停机升级

与普遍的看法相反，完全有可能在不停机的情况下升级托管数据库的计算机，借助反向代理（比如Nginx），该代理将在将存储卷从旧服务器分离并将其附加到新服务器的几秒钟内保持连接：

1.  配置反向代理将请求转发到新旧服务器；
2.  停止旧服务器上的Web应用程序；
3.  从旧服务器卸载并分离存储卷；
4.  将存储卷连接并安装到新服务器；
5.  在新服务器上启动Web应用程序；
6.  停止旧服务器。

所有这些都在不到5秒的时间内完成，反向代理保留传入连接。

![图片](https://mmbiz.qpic.cn/sz_mmbiz_png/hMIkz6ibKC9tkdsJUic78ugWKMINXyntoHh1tFQSRL3DHHI9CeDnuW2mms6JRP3qIEJXFv6JtwWFP3hVILJkVYgw/640?wx_fmt=png&from=appmsg&tp=webp&wxfrom=5&wx_lazy=1&wx_co=1)

### 零停机故障转移

但是，如果机器意外死机或数据中心烧毁，该怎么办？即使是阿里云这样世界级的云运营商在光缆被锄头砍断的情况下也变得束手无策。关键是其故障转移的成本太大，难以在短时间实现。但是对于SQLite来说实现5分钟的故障转移则很轻松：

只需要位于2个不同数据中心的2台服务器，最好是位于2个不同提供商的服务器。

当监控系统检测到主服务器停机X分钟时：

在另一个数据中心的故障转移服务器上提取最新的数据库；

在故障转移服务器上启动Web应用程序；

如果需要，更新DNS记录以指向故障转移服务器。

仅此而已。

做一个数据对比：假设允许99.99%的可用性，则每年可以有52.60分钟的停机时间。如果故障转移需要5分钟进行切换，那么每年可以有10台机器死掉，但仍具有99.99%的标签。

现实情况是，如果管理正确，现代机器比这要可靠得多，并且只要不推送任何损坏的版本，就可以轻松达到99.999%（“五个九”）的可靠性。

无论如何，SQLite的简单性将防止发生如此多的中断 ，因此认为与当今大多数应用程序运行的kafka架构相比，5分钟的故障转移风险很小。

## SQLite的陷阱

### 更新架构可能会很慢

从SQLite 3.35 开始可以支持：

```sql
ALTER TABLE DROP COLUMN
```

当表很大时，它通常会很慢。

### 无时间戳类型

SQLite缺乏一个TIMESTAMP类型，如果需要使用Unix时间戳作为INT或 ISO-8601/RFC-3339时间戳为TEXT。

一般可以使用Unix毫秒时间戳Time提供最佳性能的类型。

```go
type Time int64
func (t *Time) Scan(val any) (err error) {
switch v := val.(type) {
case int64:
*t = Time(v)
return nil
default:
return fmt.Errorf("Time.Scan: Unsupported type: %T", v)
}
}
}
return *t, nil
func (t *Time) Value() (driver.Value, error) {}
```

### COUNT（）查询速度缓慢

与PostgreSQL不同，SQLite不保留有关其索引的统计信息，因此COUNT查询速度很慢，即使使用WHERE索引字段上的子句：SQLite必须扫描所有匹配的记录。

针对这个问题，可以使用触发器方案，通过触发器在INSERT和DELETE更新单独表中的运行计数，然后查询该单独表以查找最新计数。

### 分布式SQLite

未来SQLite可能支持分布式架构，这是比较令人期待的一个方向。这些项目的基本思想大致相同：

有一个用于写入的数据库，并将其复制到“无限”数量的只读副本。

它不仅允许扩展读取，而且也许更重要的是，可以将读取副本分发到世界各地。

主数据库位于单个区域，并且在边缘进行复制以实现超快速读取。 假设Web应用程序的全球响应时间为50毫秒。

![图片](https://mmbiz.qpic.cn/sz_mmbiz_png/hMIkz6ibKC9tkdsJUic78ugWKMINXyntoHUPHCGrT1K34PHDaCJfnKrUqYCiaCn0qKAKwRu90rV0v4h9mkC0RXKkA/640?wx_fmt=png&from=appmsg&tp=webp&wxfrom=5&wx_lazy=1&wx_co=1)

该类项目有：

Cloudflare的D1、litefs（github/superfly/litefs）、（rqlite/rqlite）、mvsqlite（github/losfair/mvsqlite）、chiselstore（github/chiselstrike/chiselstore）、dqlite（github/canonical/dqlite）、cr-sqlite（github/vlcn-io/cr-sqlite）

大家可以按需尝试。

## 总结

SQLite是应具有非常广泛应用，且有无限可能的数据库。他不仅仅是你认为那个文件数据而已，你的90%的应用都可以使用SQLite数据，而且比其他数据更安全、更简便、更可靠。只需优化一下你的配置，下面是配置总结：

```sql
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;
PRAGMA cache_size = 1000000000;
PRAGMA foreign_keys = true;
PRAGMA temp_store = memory;
使用BEGIN IMMEDIATE事务；
writeDB.SetMaxOpenConns(1)；
readDB.SetMaxOpenConns(max(4, runtime.NumCPU()))；
使用STRICT表。
```
