# rc44 Canvas 图片渠道整合回执

2026-10-16 11:46 北京时间受控发布。员工刷新 https://canvas.dongyuanhub.com/ 后使用「AI生图 / 改图」，默认3:4、标准质量；添加1或2张参考图即可改图。当前只有齐东 Flare 生效。Owner 本轮只批准齐东三次实测，明确其他供应商不用测，因此 Packy 保留登记但关闭，HeyRoute/APIMart不开放自动备用。原任务的完整多供应商自动主备目标尚未上线。

## 来源、线上身份与调用链

- Canvas源码 `integration/newapi-images`，实现提交 `3463f2ff79b51377097eac07f932c9f264380d39`；从已上线历史账单 `aecaff0` 继承，合入安全候选 `ec8afa3`，保留企业v0.20.0前端。未以原作者纯前端main替换企业后端。
- New API源码 `integration/canvas-images-rc44`，实现提交 `221e1b4415bc429fe79a5793bc28cdd9dde100ff`；精确基于官方rc44 `633b4c5b190dc1af99a2814391354d86948e7943`，不是以更新main替换rc44。
- 实际链路：员工 → Canvas企业API/图片worker → `new-api-personal:3000/v1` → 分组 `canvas-image` / 模型别名 `canvas-image` → 渠道2 `canvas-qidong-flare` → `gpt-image-2.5-flare`。生成JSON与编辑multipart均通过现有OpenAI适配；响应实际为1086×1448 PNG（3:4）。
- API `infinite-canvas-api:newapi-images-20261016`，SHA `0c52a756f4b2334a57703ea02a1c65b7ac01eb3bbe91abaa6b1db6b83dfad53e`。
- Web `infinite-canvas:newapi-images-20261016`，SHA `b2e675e8ff56d6ede8b18b15ebb9bc5106799f6950224337c6f0e2f3d76d280d`。
- New API `new-api:rc44-canvas-images-20261016`，SHA `3bc14a11c4b90bd9e913e4b26be7b5c5c5f2c15514352d6d8c80a03a674c3138`；为本项目补丁版rc44。
- API按已部署账单镜像增量替换12个已验证模块和0014/0015迁移；其余原模块保留。构建输入哈希见 evidence/newapi-20261016/candidate-manifest.json；不要直接用该源码全栈重建冒充当前增量镜像。

## 当前策略与费用

仅New API做图片供应商调度，Canvas网关模型绑定唯一渠道，不在Canvas再次换商。全局RetryTimes仍0；图片分组单独有限策略。主渠道正常优先100；Packy候选优先10、状态手动关闭。已知失败冷却5分钟，后续请求绕过冷却渠道，到期按请求恢复优先主渠道，不执行付费定时探活；本次未开启Packy，所以当前主渠道失败会明确失败/待核，不会自动获得备用成功。

仅明确401或预受理错误白名单允许一次备用尝试；5xx、429、403、超时、断连、异常响应不在同一任务再次付费。无无限重试。输入配置限定单次n=1、768×1024、最多2参考；Canvas批量仍为独立任务。Flare不发送不适用的quality；Packy候选low但尚未开放。新模型没有把0.03元写成现金实付价格。

项目用户3与生产Token5均有限，发布时剩455000平台积分；以当前15000积分/次约还能30次，耗尽需Owner补充内部额度。供应商Key原有20元有限额度不变。这是已有余额的内部额度配置，不是充值或新增付款。测试Token4本轮用完，不重置、不作生产Token。

每次记录Canvas task_id、New API gateway_request_id、供应商upstream_request_id、实际渠道/模型、状态与平台quota。30秒只读同步精确请求日志，替换快照而不累加。齐东账单按唯一请求号关联后显示「供应商账户扣费」，没有可靠账单时为待核实；充值折扣/银行现金成本仍未知。旧billed_amount和历史档案未覆盖，不能将平台积分直接称为人民币。

## 实际验收与状态

| 状态 | 结果 |
| --- | --- |
| 已开发 | 最小网关接入、唯一调度层、有限故障/恢复、请求关联和费用同步；继承历史账单 |
| 已测试 | 真正Canvas worker→个人New API→齐东：生成、单参考编辑、双参考编辑各1次，共3次成功，每次1086×1448；请求号与供应商账单逐笔吻合，账户扣费各CNY0.03，共0.09，无其他供应商付费发送 |
| 已上线 | 正式Canvas API/Web和个人New API补丁；生产加密Token连通目录200，匿名账单401，首页/health200；400历史档案和412日志/261任务/227图片保持 |
| 真实业务通过 | 尚待Owner/员工实际使用验收。上述三次是真实图片验收，发生于原库副本，未伪造为员工正式业务；正式配置未额外发送第四次付费测试 |

图片实际目视：生成红杯、单图改蓝杯、双参考红蓝两杯及中文标题均吻合。证据 results.json / settlements.json 含三个Canvas任务号、两层请求号与唯一账单；真实图片保存在服务器 paid-acceptance 以及本地 integration-20261016/paid-acceptance。脚本最后退出登录204被当JSON解析导致进程退出1，三次图片和结算在此前均已落盘成功，未因此重放。

免费原生HTTP模拟9场景通过、0真实供应商请求：健康主选、503当前不重放/后续冷却绕过、主恢复优先、401有限备用、单/双参考编辑、断连不重放。模拟仅证明调度机制；Packy实际生成/费用未在本轮验收。API套件82通过/1实PG项跳过；随后针对性14项通过，TypeScript和前端构建通过；本轮真实PG副本迁移和实际worker成功。New API定向单元与真实二进制模拟通过。

## 凭据、部署文件与恢复

Key只记录位置，不记明文：齐东本地DPAPI `D:/AI/new api/secrets/qidong-image-1016.dpapi`；服务器 root0700目录下 `secrets/qidong-image.key`、`secrets/canvas-image-production.token`，生产Canvas渠道encrypted_api_key；New APISQLite渠道/Token原生管理。服务器本轮根目录 `/opt/canvas-newapi-integration-20261016`，秘密文件0600；管理员凭据仍在 `/opt/new-api/secrets/admin.json`。复用原12项渠道登记与Packy共用Key，未轮换共享Key。

原始生产DB备份 `backup-before/canvas.dump`，个人网关在线SQLite备份 `backup-before/new-api.db`；临发布新增 `release/canvas-before-release.dump`，配置 `release/before-private.json`，旧模型发布状态 `release/before-model-status.json`。原历史账单挂载 `/opt/canvas-billing-import-20261016/imports:/app/private-imports:ro` 及SHA环境均保留。新迁移仅增字段/索引，不删旧表。

恢复先停止入口新提交、等待queued/running=0，再运行本轮 `rollback_canvas.py`：恢复旧模型发布状态，回切保留的 `canvas-migrated-api-pre-newapi-20261016` 与 `canvas-migrated-app-pre-newapi-20261016`；当前容器重命名保留，新增记录/schema均保留，不用旧DB覆盖新业务。该恢复脚本已保存未在发布后执行。个人New API原容器保留为 `new-api-personal-pre-canvas-20261016`，仅需整体撤回个人网关时在Canvas先回退后再切回，rc39不参与。

后续Compose操作必须最后叠加 `/opt/canvas-newapi-integration-20261016/compose.canvas-override.json` 和个人网关 `compose.newapi-override.json`。Canvas分别沿用容器label记录的原迁移/前端/历史账单文件，再加本轮覆盖；禁止只跑旧基础compose覆盖本轮。发布保留原HostConfig/env/ports/mounts/networks，数据库/MinIO、rc39、Studio试用PG容器ID与启动时间未变；没有触碰Studio应用。临时验收两容器已停止，数据库、图片、证据和所有回滚备份保留。

## 剩余事项

员工真实使用确认；Packy费用边界和真实编辑验收后才能开启自动备用；HeyRoute/APIMart仍观察。供应商账单若短暂未出现则保持待核；目前自动核账在平台结算后停止，迟到的供应商账单还需补一次只读复核。历史400条仍272候选/128待核，未伪装完成真实实付。当前不是全供应商、所有尺寸/质量均验证完毕的发布。
