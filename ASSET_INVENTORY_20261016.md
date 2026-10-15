# 服务器资产只读盘点（2026-10-16）

范围为 Canvas / New API 及同机依赖识别，不是全服务器清理。通过本机 SSH 检查，无 RDC。未删除目录、镜像、数据卷或数据库；升级新增及停止的测试容器也保留。状态含义：正式使用=当前入口或权威数据；仍有依赖=回滚或现存进程/配置依赖；确认废弃=本次可证明已结束的临时验证用途；用途不明=尚不能证明可删。停止/无标签/无容器引用均不自动等于废弃。

## 运行和停止容器（全部 11 个）

| 容器 | 状态/项目 | 域名、端口、版本和数据来源 | 分类 |
| --- | --- | --- | --- |
| canvas-migrated-app | running / Canvas | canvas.dongyuanhub.com →127.0.0.1:13001；前端 v0.20.0，镜像 7e6d6c1575fb；无业务卷 | 正式使用 |
| canvas-migrated-api | running / Canvas | 内部3001，原镜像8d467b14cf09；企业 v0.18.0 部署来源，账户/任务/渠道调用 | 正式使用 |
| canvas-migrated-postgres | running / Canvas | 内部5432；PostgreSQL17.11，18cfe3ef5e68；volume canvas-migrated-20260919_postgres_data | 正式使用 |
| canvas-migrated-minio | running / Canvas | 内部9000；RELEASE.2025-07-23T15-54-02Z，d249d1fb6966；volume canvas-migrated-20260919_minio_data | 正式使用 |
| new-api-personal | running / New API | newapi.dongyuanhub.com →127.0.0.1:3044；v1.0.0-rc.44，67bf27160217；/opt/new-api/data、logs bind | 正式使用 |
| sw-new-api-rc39 | running / ScienceWorkbench | 127.0.0.1:3039；v1.0.0-rc.39，219b0cecfd7d；/opt/scienceworkbench/private/new-api-rc39/{data,logs} bind | 仍有依赖 |
| dongyuan-trial-postgres | running / 东元 Studio | 127.0.0.1:18432；PG17.11，18cfe3ef5e68；/var/lib/dongyuan-studio/postgres/data bind | 正式使用，保护 |
| canvas-migrated-app-rollback-20261016 | stopped / Canvas旧正式app | 镜像6753f0800c94，原容器c0f2b4925917，已实际回切验证 | 仍有依赖，回滚保护 |
| canvas-upgrade-candidate-r2-20261016 | stopped / 本次候选 | 镜像7e6d6c1575fb，与当前正式镜像共享，无业务卷 | 确认废弃的测试容器；镜像仍正式使用 |
| canvas-upgrade-candidate-20261016 | stopped / 本次首候选 | 镜像4ca1641343e1，静态权限403问题，未正式发布，无业务卷 | 确认废弃的失败候选 |
| canvas-upgrade-restore-20261016 | stopped / 本次恢复测试 | 镜像18cfe3ef5e68，network none、tmpfs，无持久化业务卷 | 确认废弃的验证容器；镜像仍正式使用 |

没有第二套运行或停止的旧 Canvas 全栈容器。宿主机进程列表里的 MinIO/Node 已通过 cgroup 归属上述 Docker 容器，不重复计为一套原生 Canvas。

## 数据库、卷和备份

| 数据 | 体积 / 已查内容 | 分类与保护理由 |
| --- | --- | --- |
| Canvas PG volume | Docker约78.46 MB，du约75 MiB；12核心表恢复指纹一致；升级前 batches247/tasks258/images227/request_logs409 | 正式使用；账号/画布/历史/渠道权威数据 |
| Canvas MinIO volume | Docker约932.1 MB，du约896 MiB | 正式使用；用户图片和素材，不可 prune |
| New API rc44 SQLite | /opt/new-api/data/new-api.db，933888 B；2用户、3token、1 DeepSeek渠道、7日志（6消费+1充值） | 正式使用；渠道/计费数据 |
| New API rc39 SQLite | /opt/scienceworkbench/private/new-api-rc39/data/new-api.db，1626112 B；1用户、1token、1 DeepSeek渠道、578消费日志 | 仍有依赖；不能因更旧而删除 |
| Studio PG bind | /var/lib/dongyuan-studio/postgres/data，约690 MiB | 正式使用；禁止修改/清理 |
| 原生 MariaDB | /var/lib/mysql约531 MiB；业务库 sw_employee_p0，另系统库 | 仍有依赖；旧 Frappe 服务运行，具体业务保留规则未核清 |
| 原生 Redis | /var/lib/redis约3.4 MiB；db0共107key | 仍有依赖；未证明可清空 |
| 本次完整回滚备份 | /opt/canvas-upgrade-20261016/backup约996 MiB | 仍有依赖；PG、MinIO、源/WIP/配置、镜像，禁止删 |
| Windows浏览器/DB备份 | 本任务根backups，受保护；origin导出3.39MB、原IDB约2.27MB、另数据库dump | 仍有依赖；禁止上传公开仓库或随清理删除 |

仅有两个 Docker named volume，全部使用中，Docker报告可回收卷空间0。其余是bind，不能因 `docker volume ls` 没列出而漏判数据库。新增验收是1次失败任务，request_logs从409变410；不是数据丢失。完整业务/账户内容仅存私有证据，不公开。

## 原生服务与监听

| 服务/进程 | 入口/来源 | 分类 |
| --- | --- | --- |
| nginx | 80/443；正式域名仅canvas、新api、work三条反代；Canvas13001/NewAPI3044/Studio18420 | 正式使用 |
| dongyuan-trial.service | work.dongyuanhub.com；/opt/dongyuan-studio/releases/d43089ffe4a17a85c9c8316fe066a038d0f1f2cf，Node24.21.0，127.0.0.1:18420 | 正式使用，保护；health=200 |
| dongyuan-page-service.service | /var/lib/dongyuan-studio/page-service/release-r2，Python3.12.15，127.0.0.1:18488 | 正式使用，保护 |
| sw-local-web.service / kernel | /opt/sw-local/bench/frappe-bench、/opt/sw-local/kernel；Python3.11.15，127.0.0.1:18200/18202 | 仍有依赖；旧系统在运行，版本业务归属待核 |
| 其他 Node 监听 | 127.0.0.1:18320/18321、18468/18469；/opt/scienceworkbench 与 scienceworkbench-next 的现存进程 | 仍有依赖；精确业务版本未核清，不可按目录名删 |
| Temporal | 127.0.0.1:17233/17234及动态本地端口；/opt/sw-local/temporal等 | 仍有依赖；服务消费者未完全核清 |
| mariadb / redis-server | 127.0.0.1:3306；127.0.0.1及::1:6379 | 仍有依赖 |
| new-api-backup.timer / service | timer活动；service轮次间inactive；/opt/new-api脚本与备份 | 仍有依赖；inactive不是废弃 |

## 历史目录和实际空间

以下为du同一时点近似，父子目录不可相加；未逐个验证所有历史release的版本和业务来源，未核清项标用途不明，不能编造为已废弃。

| 路径 | 实际占用 | 分类/依赖 |
| --- | --- | --- |
| /opt/dongyuan-studio | 36.38 GiB，其中releases36.24 GiB | 当前release正式使用，其他历史release用途不明/可能回滚；全部暂保护 |
| /opt/scienceworkbench | 19.29 GiB，其中private10.70GiB、recovery6.82GiB、dev1.66GiB | 混合运行依赖/历史部署；整体不可删 |
| /opt/scienceworkbench/canvas | 约15MiB，原Git/WIP/配置 | 仍有依赖，原正式来源 |
| /opt/scienceworkbench/recovery/canvas-migration-20260919及.transfer.tar | 各约327MiB | 用途不明/迁移回滚来源，数据唯一性未比对 |
| /opt/scienceworkbench/private/canvas-task-bridge-deploy-20260920 | 约460KiB，new.json仍在Compose config_files引用 | 仍有依赖 |
| /opt/scienceworkbench/private/canvas-simple-deploy-20260921 | 约18MiB | 用途不明，历史实验配置可能含凭据 |
| /opt/scienceworkbench/private/canvas-padding-deploy-20260921 | 约3.7MiB | 用途不明，历史实验配置 |
| /opt/scienceworkbench/private/new-api-rc39 | 当前DB/日志/私有配置 | 仍有依赖 |
| /opt/scienceworkbench-next | 4.36GiB；backups1.38GiB/toolchain1.21GiB/home1.01GiB/var520MiB/dev199MiB | 有现存进程；备份/工具链用途未完全核清 |
| /opt/sw-local | 1.57GiB；bench979MiB，其他Python/Node/Temporal/backup | 仍有依赖，旧原生服务运行 |
| /opt/new-api | 约2.3MiB | 正式使用 |
| /opt/canvas-upgrade-20261016 | 约1GiB含完整备份及本次release | 仍有依赖，回滚保护 |
| /opt/scienceworkbench-html-extractor-2.2.0 | 约154MiB | 用途不明/可能运行依赖 |
| /opt/scienceworkbench-recovery-58fd-20260926 | 约12KiB | 用途不明 |

## 镜像清单（共29个image ID）

相同image多tag只算一份；无tag也可能正式使用。29个ID的磁盘共享层合计4.128GB，不能把每行逻辑SIZE相加。

| image ID前缀 / tag | SIZE | 分类 |
| --- | --- | --- |
| 7e6d6c1575fb / enterprise-v020-20261016-r2 | 93.2MB | 正式使用 |
| 4ca1641343e1 / enterprise-v020-20261016 | 88.3MB | 确认废弃首候选，仍被停止容器引用 |
| 6753f0800c94 / task-bridge-20260920-r1、canvas-upgrade-base:20261016 | 83.4MB | 仍有依赖，回滚和新镜像父层 |
| 8d467b14cf09 / 无tag | 437MB | 正式使用Canvas API |
| 18cfe3ef5e68 / 无tag | 424MB | 正式使用PG17.11，两个生产容器引用 |
| d249d1fb6966 / 无tag | 241MB | 正式使用MinIO |
| 67bf27160217 / calciumion/new-api:v1.0.0-rc.44 | 325MB | 正式使用 |
| 219b0cecfd7d / calciumion/new-api无tag | 328MB | 仍有依赖rc39 |
| 96af8d2dc39f / canvas-title-build:20260922 | 1.19GB | 用途不明历史构建，0容器 |
| 5669eace8d26 / canvas-simple-build:20260921 | 1.19GB | 用途不明历史构建，0容器 |
| 99f8fa2e3e6b / canvas-padding-build:20260921 | 1.19GB | 用途不明历史构建，0容器 |
| bfcc37d1384f / scienceworkbench/canvas:padding-20260921 | 88.2MB | 用途不明历史前端，0容器 |
| f95f817514a6、bc7d2e9e2ea5 / spa-cache-fix两版 | 各78.5MB | 用途不明历史前端，0容器 |
| 3bf8bf8e0707 / batch-clipboard-20260920 | 78.5MB | 用途不明历史前端，0容器 |
| 589320917385、d2a56d14488f、c417ee1fe9b5 / employee-batch-repair-r2/r1、employee-ux-r1 | 各78.5MB | 用途不明历史前端，0容器 |
| 0a86d93866c5、ec517e917cfb、c0ee86f336bd、129fd3c2d7e0、28d58885a170 / 无tag | 各1.19GB | 用途不明历史中间构建，0容器；大量共享层 |
| cbd60983fdd7 / 无tag | 98MB | 用途不明历史镜像，0容器 |
| 053081b987ce / 无tag | 78.5MB | 用途不明历史镜像，0容器 |
| 5a5a84b19854 / postgres:latest | 650MB | 用途不明，0容器，不是当前PG镜像，具体版本未运行核验 |
| 87416c977a61 / oven/bun:1.3.13 | 324MB | 历史构建工具，后续构建可能依赖 |
| 93435df079cf / mariadb无tag | 431MB | 用途不明，0容器；原生MariaDB仍在运行，不能混为同一数据 |
| 65645c7bb6a0 / nginx:1.27-alpine | 74.5MB | 构建父层/历史依赖，0直接容器 |

## 资源快照与清理建议（未执行）

01:16 CST快照：根盘190109507584 B总、122939383808 B已用、59315458048 B可用，68%，inode30%；内存15.12GiB总、5.90GiB已用、8.79GiB available、无swap。盘点时CPU采样us26%/sy17%/idle45%/I/O wait12%，当时有备份/构建，不能当作长期基线。旧kernel Python约1.42GiB RSS、gunicorn两worker约0.98GiB总、MariaDB约614MiB，是重要内存来源。容器快照：API228MiB、MinIO277MiB、StudioPG130MiB、CanvasPG40MiB、两NewAPI各34–36MiB、原app12MiB；均非长期峰值。

Docker29镜像合计4.128GB，报告reclaimable989MB是**引擎口径上界**，不是允许删除的清单；两个卷约1.012GB，0可回收；11容器写层15.56MB，其中停止容器约15.34MB，绝大部分是必须保留的旧回滚app。build cache=0。

1. 可以请Owner统一确认清理的低风险项目：本次首候选/r2测试容器、tmpfs恢复测试容器；无业务卷，容器本身收益仅KiB级，首候选镜像少量独有静态层。保留正式r2镜像和旧回滚app；不能以停止容器15.34MB为可释放承诺。
2. 历史未引用的Canvas build镜像、旧前端、闲置postgres:latest可形成下一批**待核准**清单。最明确的postgres:latest独有650.1MB，但仍需确认未来用途；三个1.19GB build各报告约214MB unique，不能机械累计为3.57GB。整个镜像集合最多按当前989MB报告作参考，实际删除需逐ID计算共享层并保留新旧发布及构建来源。
3. 目录收益可能更大，但Studio releases、ScienceWorkbench private/recovery、next backups尚不能证明废弃，预计可安全释放空间UNKNOWN。禁止为腾空间直接扫删这些目录。先限定版本/数据唯一性、运行进程和回滚保留窗口，再单独出清理单。

升级及真实回滚已验证；本阶段只提交建议，不执行任何删除。下一轮清理必须由Owner确认精确项目/路径/ID和备份保留范围，禁止 blanket prune、volume prune 或递归清理整个项目树。
