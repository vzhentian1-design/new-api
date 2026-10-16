# 当前生效：Flare / Sunburst 独立自动主备已上线

员工只显示 GPT Image 2.5 Flare（默认）与 GPT Image 2.5 Sunburst；两款唯一绑定个人 New API，各自齐东100 / Packy10，冷却300秒、实际新请求成功恢复，最多两次、主180/备240秒，Canvas不做第二层付费重试。正式首页/健康接口200、模型选择器及四渠道配置已读回；原400历史账单、412日志/261任务/227图片保持，rc39/Studio不改。

Owner先批2次：齐东Sunburst编辑成功并核CNY0.03，PackyFlare因response_format返回400后立即停止；仅在另批1次后补测，修复后的PackyFlare在假主渠道401后真实成功，39753平台积分、供应商实际扣费待核实。累计真实发送3次、成功2图；旧齐东Flare双参考与PackySunburst历史编辑复用，修复后24免费场景通过。不再付费补测。

最终镜像：API b0ad358fb014、Web 8823808ae9ec、New API adbd3f91f50b（实现5d297bd）。私有发布 /opt/canvas-newapi-autofailover-20261016/release-r2；三个前生产容器以 -pre-autofailover-20261016-r2 保留。已实际处理一次发布回滚，最终r2上线；回滚脚本默认检查PASS，只回切配置/容器、不回灌业务库。所有本轮验收容器停止。

当前未完成：Packy真实逐笔扣费关联、Owner/员工日常业务确认。完整事实以本阶段发布结果为准，旧成功与原回滚资产保留。

## 以下为本轮发布前的历史交接，当前状态以上方及发布结果为准
# 当前阶段：自动主备候选通过免费验证，等待付费预算及边界确认

正式链路继续使用已上线 Canvas → 个人 New API rc44 补丁 → 齐东 Flare，旧三次生图/单参考/双参考及逐笔扣费证据保留。Packy 正式关闭；本轮无新付费请求、无生产发布，不修改 rc39/Studio。

当前候选、24 个模拟场景、三种真实数据库、逐渠道计费与 Packy 账单缺请求号的限制见 [CANVAS_AUTOMATIC_FAILOVER_20261016.md](CANVAS_AUTOMATIC_FAILOVER_20261016.md)。Owner 已撤销三次双参考计划，最少只补齐东 Sunburst、Packy Flare 各一次单参考编辑，后者合并验证假主渠道失败后的真实备用完成；Packy Sunburst 35 条历史编辑记录复用。待批准 CNY 0.03 + USD 3.56 风险预留，不另加付费测试。候选分支仍为 `integration/canvas-images-rc44`，Canvas 为 `integration/newapi-images`。不得将候选状态当上线，不重用旧预算，不重启任何已收口的旧付费验收脚本。

下一步：Owner 批准预算/新运行边界后，在现有生产基础上做最小付费验证，再按已保留配置和镜像安全发布/回滚。主 180 秒、备 240 秒、每款冷却 300 秒、最多两次；Packy low 与真实扣费未知需明确保留。现有全局 RetryTimes=0 和 Canvas 唯一发送保持。

本轮服务器私有工作目录 `/opt/canvas-newapi-autofailover-20261016`；生产镜像未回切，全部本轮测试容器停止。候选镜像不能被旧基础 Compose 静默覆盖；原正式部署/备份/回滚仍见 [上一轮发布回执](CANVAS_IMAGES_20261016.md)。
