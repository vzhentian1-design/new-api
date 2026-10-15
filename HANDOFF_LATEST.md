# New API 接入核验交接（2026-10-16）

本轮是只读核验及文档归档，没有更改 New API 代码、版本、token、渠道、重试或计费配置。Canvas 企业 v0.20.0 前端已升级并完成真实回滚演练；真实生图首个请求供应商401，改图未发送。完整升级记录在 [Canvas 交接](https://github.com/vzhentian1-design/infinite-canvas/blob/upgrade/enterprise-v020/HANDOFF_LATEST.md)。本分支仅文档，未合并 main。

## 当前真实实例

| 实例 | 来源、入口和数据 | 分类 |
| --- | --- | --- |
| new-api-personal | calciumion/new-api:v1.0.0-rc.44，镜像67bf27160217；https://newapi.dongyuanhub.com →127.0.0.1:3044；SQLite /opt/new-api/data/new-api.db | 正式使用；2用户、3token、1个DeepSeek文本渠道、7条日志（6消费+1充值）；RetryTimes=0 |
| sw-new-api-rc39 | 运行命令version=v1.0.0-rc.39，镜像219b0cecfd7d；127.0.0.1:3039；SQLite /opt/scienceworkbench/private/new-api-rc39/data/new-api.db | ScienceWorkbench仍有依赖；1用户/1token/1个DeepSeek文本渠道/578消费日志，禁止按旧版残留删除 |

维护Fork main当前提交 `63cecab824bad1a421d19950004c309bb2054c09` 比线上新。匹配rc44的代码tag为 `v1.0.0-rc.44` / `633b4c5b190dc1af99a2814391354d86948e7943`；以下代码结论均以该tag为准。两个实例当前均没有图片渠道，不能将其文本日志作为Canvas图片费用证明。容器ID、启动时间、镜像与配置在本轮Canvas切换后均比对不变。

## Canvas 链路与供应商事实

浏览器 → Canvas企业API/worker → Canvas PostgreSQL model_channels/channels → **直接** `https://api.qidong.autos/v1` 或 `https://cf.api.fan/v1`。尚未经过以上两台New API。

qidong使用upstream `gpt-image-2-qidong`，Packy使用 `gpt-image-2` / `gpt-image-2.5-flare` / `gpt-image-2.5-sunburst`；现有每个模型只有一个渠道，priority都为0，所以不是一主一备。Canvas自己已有优先级和冷却调度能力，但此配置下没有第二候选。qidong已有真实503记录（例如2026-10-15 05:43:11 UTC的request_log `946bbf06-c39c-4f97-93bd-f538240538bb`），供应商“active”不等于健康或恢复。

最新唯一验收调用（2026-10-15 17:10:33 UTC）返回401 Invalid token，Canvas task `d418214e-613a-48ab-af5a-12b27827cc46`、log `c5b2f25e-edee-4d12-864b-07e517c68c29`、供应商请求号 `202610151710337498450728268d9d6qILNmZpO`，同task仅1个attempt。已停止，无改图、无Packy验收、无自动重试。billed_amount=NULL不等于现金0。

Owner给出齐东¥0.03/张，当前登录供应商模型广场搜索gpt-image-2没有结果，当日请求号日志无匹配记录；令牌列表有禁用项，但未揭示/匹配服务器密钥，无法进一步断言是哪一个失效。Packy当前页面将三个图片模型列为image分组按次计费、$0.029–$3.558范围、显示7.2折，不能继续无条件使用“按输入输出计费”的旧假设。本轮没有生成成功对应的供应商现金账单；单价适用性/扣费仍UNKNOWN。

## 已有主备、故障和恢复接口

| 能力 | rc44真实代码/默认值 | 对本项目的含义 |
| --- | --- | --- |
| 优先级和权重 | model/ability.go：priority降序、同级权重，重试索引选后续优先级 | 可复用，不必另造一套路由器 |
| 请求重试/降级 | service/relay_error.go DecideRelayRetry；common.RetryTimes=0，线上配置0 | 当前普通503不自动增加供应商发送 |
| 状态范围 | setting/operation_setting/status_code_ranges.go：重试默认500–503，排除504/524；自动禁用默认401 | 可配置范围不等于开关已启用 |
| 自动禁用/恢复 | service/channel.go ShouldDisableChannel/ShouldEnableChannel，默认开关false；成功探测可恢复已自动禁用渠道 | 不等于当前已有自动恢复运行证据 |
| 渠道测试 | controller/channel-test.go settleTestQuota | 会调用上游并结算，不能当免费探活或自动打开 |

难度判断：复用能力中等、主要工作是模型映射、隔离配置和费用边界验证；无需先重写网关。最小阶段二先处理qidong凭据/模型有效性并完成受限生图改图验收，再在两条路中选择一条：A复用Canvas多渠道映射（较快）；B统一到独立rc44网关（新增隔离图片token/分组和映射），Canvas只保留一个网关渠道。禁止同时在两层开重试，禁止动rc39和Studio。503/断连可能上游已受理，须有预算、幂等、UNKNOWN和人工恢复规则后另行裁决付费降级；本轮未实施。

## 逐请求计费接口：可用，但不能冒充现金成本

rc44 router/api-router.go：

- `GET /api/log/`：AdminAuth，管理日志，支持request_id筛选。
- `GET /api/log/self`：UserAuth，当前用户日志。
- `GET /api/log/token`：TokenAuthReadOnly；model.GetLogByTokenId按token过滤并限制最近条数，不保证无限历史。
- common/constants.go：响应头请求号 `X-Oneapi-Request-Id`、上游请求号 `X-Upstream-Request-Id`。middleware/request-id.go生成网关请求号；应保存响应头做关联，不能假定客户端任意ID会被沿用。

实际线上用已有保护配置中的管理员身份正常登录后，`GET /api/log/?p=1&page_size=5`返回HTTP200/success，5条已存在DeepSeek记录的quota为14/660/34/30/4；以其中现有request_id筛选返回HTTP200且匹配1条；检查会话随后logout=200。没有新建密钥、没有生成请求。最小脱敏回执在本机任务 `evidence/newapi-log-endpoint.jsonl`。`/api/log/token`和图片扣费尚未做真实请求验收，代码存在不能写成图片集成已完成。

线上 `/api/status`：quota_per_unit=500000，display_in_currency=true，usd_exchange_rate=7.3。必须独立显示：

1. **平台计费积分**：New API已结算日志quota；可按配置换算平台展示金额。
2. **供应商现金成本**：供应商对用户钱包的真实扣减，需供应商账单/成本接口/结算规则；当前没有图片真实样本。
3. **估算金额**：模型价或汇率推算。7.3是本实例显示配置，不是已核实实时汇率；充值折扣、分组倍率、退款也不能忽略。

当前Canvas `generated_images.billedAmount`来源task.priceSnapshot，即平台模型配置快照，不是供应商实际费用。不能直接改名“实际成本”。

阶段三最小实现：服务端持久保存Canvas task_id ↔ 网关响应请求号 ↔ 日志记录，结算后读取本token消费，前端显示“待结算/已结算/未知”，同时列出来源、单位和估算标记。不要下发管理员token或全站日志；近期token日志窗口需要及时同步。平台积分显示中等开发量；真实现金成本自动对账需另核供应商能力，不承诺已有接口。

## 资产、目录与文档

完整只读盘点见同仓库 `ASSET_INVENTORY_20261016.md`：7运行/4停止容器、29镜像、2活动卷，当前盘68%、可用约55.25GiB；Docker镜像报告最多989MB可回收只是口径上界。两个New API都保留，Studio正式release/数据库、Canvas数据和回滚备份全部保护。没有进行清理。后续仅由Owner确认精确删除清单再统一执行。

[New API项目入口](https://app.notion.com/p/3face9f11b9b8154bcdfcf48c824f33e) · [Canvas项目入口](https://app.notion.com/p/3face9f11b9b81bf9b80d5ec21f22085)。总目录复用既有「项目」数据库，已有东元Studio档案保留原位引用；后续每个里程碑更新版本、完成内容、证据、遗留和下一步。阶段一收口，未擅自推进阶段二/三。
