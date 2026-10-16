# Canvas 两款同型号自动主备已上线

沿用 integration/canvas-images-rc44 的个人 rc44 补丁，不升级、不改 rc39/Studio。实现提交 5d297bd；正式网关镜像 adbd3f91f50b，二进制 SHA256 9ad9b3c9968798cb388476e79b1feb673383dcb3c6ef37d0753e1597fb9ee53f。

## 当前路由

- canvas-image → 齐东渠道2 / Packy渠道3 → gpt-image-2.5-flare。
- canvas-image-sunburst → 齐东渠道4 / Packy渠道5 → gpt-image-2.5-sunburst。
- 四渠道启用；齐东优先级100、Packy10。两款独立冷却300秒，到期实际新业务请求成功才确认恢复，不定时付费探活、不跨型号替换。
- 仅 New API 调度；Canvas 单次网关发送，全局 RetryTimes=0 保持。本项目环境开关允许503/超时/断线等结果未知时有界备用，最多两次且 Packy 最后；原请求仍可能扣费。可查原请求号时先核账单，已确认扣费不再发送；内容安全、400/403/422、已开始响应及取消不盲目换商。
- Owner 已批准主180秒、备240秒；Canvas总480秒不变。Packy low/768×1024/n=1，齐东不传quality；原多参考证据复用，不新增逐型号付费双参考验收。

## 实测、修复与费用

Owner 先批最多2次真实调用：齐东 Sunburst 单参考改图成功（1086×1448、15000积分、精确供应商账单CNY0.03）；假齐东发送前失败后 Packy Flare 已接管，但400拒绝 response_format，无图，平台0积分、供应商扣费待核实，立即停测。

修复仅对本项目 Packy 的 JSON生成及multipart编辑去掉 response_format，齐东及其他项目保留原字段。Owner 单独批准再补1次 Packy Flare，同一任务假主渠道401 → 真 Packy成功（768×1024、39753积分）。本阶段总真实发送3次、成功2图，不充值、不继续补测。假主渠道没有联系真实齐东，不能称真实齐东生产故障。

修复后原生网关24场景免费回归通过，含精确模型/质量/文件数/response_format、503当前任务备用、独立冷却与新请求恢复、拒绝/参数错误、断线/超时、最多两次；相应Go测试通过。原真实SQLite/MySQL8/PostgreSQL冷却及Token隔离证据复用；未改DB逻辑。Canvas API原88通过/1原PG并发项跳过，结算另用真实PG测试；前端类型/构建通过。

齐东按张CNY0.03；Packy使用自身model_ratio2.5 × image组5 = 专用身份12.5，completion_ratio6、image_ratio1.6，不设置齐东固定三分钱。公开区间USD0.029400–3.557850不是硬封顶；主备风险按CNY0.03 + USD3.557850分别记录。生产Token5仍剩455000平台积分，未增加额度；只扩两款别名白名单。

每次记录供应商、真实模型、两层请求号、结果、平台积分、独立币种估算和可核扣费。Packy Token日志仍 record not found、网页历史样本无请求号，真实逐笔费用保持待核实；39753平台积分或共享余额变化不能冒充供应商实际扣费。现有四渠道配置、计费身份和最终平台目录免费读回通过。

复用原齐东Flare生图/单参考/双参考及逐笔账单，齐东Sunburst既有生成、PackyFlare既有生成、PackySunburst35条历史编辑。PackySunburst新网关实际生成未单独付费重测，透明保留证据边界。

## 发布与保护

Canvas共用生产选择器读回两项、Flare默认，可切换Sunburst后恢复Flare；正式浏览器会话为管理员，独立真实验证为普通用户API。首页、Web→API健康、个人网关状态均200。保留412日志/261任务/227图片和400历史账单（272候选/128待核），inflight=0。rc39、Studio试用PG、CanvasPG/MinIO的容器身份/配置/挂载/启动时间保持，验收容器停止。实际Owner/员工日常业务确认仍待。

私有发布目录 /opt/canvas-newapi-autofailover-20261016/release-r2，备份PG、SQLite与配置。前生产三个容器以 -pre-autofailover-20261016-r2 保留；Compose最后叠加本轮 compose.canvas-autofailover.json / compose.newapi-autofailover.json。rollback_autofailover.py 默认检查PASS，执行需 --execute-rollback；等活动任务归零，只恢复本项目配置和容器，不回灌数据库，不重置已耗积分。第一次发布已实际回滚并恢复反代；最终r2先验API内部健康、再替换前端，成功发布。失败诊断保留，未为演练再停一次正式服务。

[Canvas完整发布、逐次证据及回滚说明](https://github.com/vzhentian1-design/infinite-canvas/blob/integration/newapi-images/RESULT_AUTOMATIC_FAILOVER_20261016.md)。原正式发布回执 CANVAS_IMAGES_20261016.md 继续保留；新配置取代其中“只有齐东/Packy关闭”的阶段状态。

未完成仅保留：Packy真实逐笔扣费关联、Owner/员工业务确认。无待执行付费调用，不扩大双参考或付费恢复验收。