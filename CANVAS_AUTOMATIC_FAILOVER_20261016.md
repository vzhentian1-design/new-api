# 当前生效：三模型、五渠道与管理计费收口

本阶段已按 Owner 最新口径发布：金额数值不换算，内部统一显示 USD；1:1 是展示规则，不是外汇汇率。后台保留供应商原币、原金额和核实状态。齐东平台价格保持 USD 0.03/张（15000 平台积分），供应商实际原币账单为 CNY 0.03/张；Packy 使用自己的动态用量计费，未套用齐东价格。

## 员工和管理员入口

- [员工生图工作台](https://canvas.dongyuanhub.com/image)：默认 GPT Image 2.5 Flare，也可选 GPT Image 2.5 Sunburst、GPT Image 2。员工不选供应商，添加参考图可改图。
- [模型广场](https://newapi.dongyuanhub.com/pricing)、[模型管理](https://newapi.dongyuanhub.com/models/metadata)：三款真实模型 ID 和用途/路由/价格说明；管理员可查看 Canvas 专用分组目录，未扩大普通账号或 Token 的调用权限。
- [渠道管理](https://newapi.dongyuanhub.com/channels)：按 canvas 筛选五条渠道，名称/真实模型/优先级/状态及备注对应下表。状态菜单可手动禁用、启用；手动禁用不由自动冷却恢复解除。不要点击付费渠道测试来代替启用操作。
- [New API 使用日志](https://newapi.dongyuanhub.com/usage-logs/common)：渠道列是实际所用供应商；详情增加员工、Canvas 任务号、两层请求号、逐次尝试、平台积分/金额、供应商原币账单和状态。旧记录未携带员工身份则显示未记录，不伪造历史。
- [Canvas 管理员请求日志](https://canvas.dongyuanhub.com/admin/logs)：原列表直接看实际供应商、自动备用、USD 展示及详细账单；[历史账单](https://canvas.dongyuanhub.com/admin/logs?view=history)继续保留。

| 员工模型 / 真实模型 ID | 齐东渠道 | Packy 渠道 | 规则 |
|---|---|---|---|
| GPT Image 2.5 Flare / gpt-image-2.5-flare | #2 canvas-qidong-flare，100 | #3 canvas-packy-flare，10 | 默认；同型号主备 |
| GPT Image 2.5 Sunburst / gpt-image-2.5-sunburst | #4 canvas-qidong-sunburst，100 | #5 canvas-packy-sunburst，10 | 同型号主备 |
| GPT Image 2 / gpt-image-2 | 无 | #6 canvas-packy-image2，100 | 仅 Packy，无备用、不跨型号 |

Canvas 三模型均唯一绑定原个人 New API 网关，修复 Sunburst 缺失 routing=newapi 标记，保留模型 UUID 和既有 Flare/Sunburst 内部别名以维持默认选择。New API 对外目录与模型白名单使用真实模型 ID，旧别名不再作为广场标题。其他项目 DeepSeek 模型、渠道、价格不改。

## 自动切换、费用证据和未完成项

沿用此前两款 2.5 的调度器、主180/备240秒、Canvas480秒、冷却300秒及最多两次尝试；New API 唯一换商，Canvas 单次发送、不重复付费重试。冷却结束仅标待业务确认，下一次真实请求成功后确认恢复，不付费探活。既有未知结果可能仍扣费的规则保持，不把失败平台积分0解释成供应商免费。

平台金额 = 平台积分 / 500000，独立于供应商实扣。Packy 三款按已核公开规则：模型2.5 × image分组5 = 专用模型倍率12.5；输出倍率6、图片输入倍率1.6，按实际用量动态结算。上述平台金额不是供应商已确认账单。全局 USDExchangeRate=7.3 保持，不改其他项目；本项目内部展示数值按1:1标USD。

供应商原币核实需精确请求号、模型、来源、金额/币种与证据摘要；齐东已具备可靠的逐请求原币账单接口。Packy 当前 cf.api.fan/api/log/token 返回404；www.packyapi.com/api/log/token 返回HTTP200但success=false、record not found。现有导出389行中271条图片消费（Image2 225、Flare6、Sunburst40）没有请求号列且会话ID为空，无法唯一绑定Canvas任务。共享余额变化、相近时间和模型金额不作为实扣证明。保留30秒只读结算同步：精确同步New API平台积分/金额，实际扣费明确待核实，后续需供应商提供可靠逐请求账单。400条历史档案原值保持，272候选/128待核，不覆盖、不重复计费。

本轮新增付费模型调用0。Image2沿用历史真实消费及本轮免费生成/多参考编辑/失败不跨模型验证，没有新增付费全链路图片或Owner业务验收。员工详情的可选供应商字段未扩展，管理员账单入口已完成。Studio将来可能接入New API仅记录方向，本轮不迁移。

## 代码、验证与发布保护

Canvas：仅 worker 透传员工身份、gateway请求头、结算证据及管理员日志/模型/统计页；企业API以现有镜像增量覆盖所需编译模块，未以普通main替换。移除旧统计页把未注明币种金额作为人民币汇总的误导展示，明细保留。New API：精确三模型路由约束、Image2仅Packy、日志身份和展示规则、原生详情中文字段及冷却状态说明；管理员模型广场目录补齐。无schema变更。

定向验证：Canvas结算6项和网关单次发送/员工身份2项；New API精确模型限制Go测试、原生日志UI8项与渠道状态已有测试；两前端类型检查/构建、API编译、GoLinux构建通过。最终原生网关8场景用假密钥/本地上游验证Image2生成/多参考编辑/失败不换型号、两2.5真实ID、同型号备用/冷却/恢复及逐次积分。免费模拟不冒充真实供应商质量或业务验收；此前24场景和三数据库证据复用，不重复大套测试。

生产412日志/261任务/227图片及6业务表指纹保持，400历史账单保持；生产Token余额455000、平台用户余额及消耗/错误日志未变。Canvas PG/MinIO、旧rc39与Studio试用PG容器身份/配置/挂载/启动时间保持。无待执行付费测试，免费验收容器停止。

私有发布证据 `/opt/canvas-management-closeout-20261016/release/`，含原始PG/SQLite备份、配置/容器快照、发布/价格纠正/展示与目录回执。初版错误地按7.3换算平台价格，Owner纠正后已恢复0.03，期间未新增消费；最终按Owner口径1:1展示。早期回执的0.004110与首次镜像仅为中间状态，不作为当前配置。

当前最终API镜像 `b549c23da987`，Web `03862fd73855`；最终网关 `bbdeeb47de90`（完整ID以同目录 `catalog-completed.json` 为准）。发布镜像标签分别为 `infinite-canvas-api:management-20261016`、`infinite-canvas:management-20261016`、`new-api:canvas-management-20261016`。

原三个容器以 `-pre-management-20261016` 保留；展示/目录修正前容器另保留，不覆盖旧阶段回滚。运行 `sudo python3 /opt/canvas-management-closeout-20261016/release.py` 只检查回滚资产；显式 `--rollback` 才回切原配置与容器。回滚不回灌业务库、不重置积分，不覆盖新任务；需先确认无进行中任务。当前阶段收口，不创建后继总控、不继续扩展平台。

---

以下是上一阶段记录，仅作历史；当前配置以上方为准。

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