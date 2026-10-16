# 当前阶段：自动主备候选通过免费验证，等待付费预算及边界确认

正式链路继续使用已上线 Canvas → 个人 New API rc44 补丁 → 齐东 Flare，旧三次生图/单参考/双参考及逐笔扣费证据保留。Packy 正式关闭；本轮无新付费请求、无生产发布，不修改 rc39/Studio。

当前候选、24 个模拟场景、三种真实数据库、逐渠道计费与 Packy 账单缺请求号的限制，以及 3 次最小测试预算见 [CANVAS_AUTOMATIC_FAILOVER_20261016.md](CANVAS_AUTOMATIC_FAILOVER_20261016.md)。候选分支仍为 `integration/canvas-images-rc44`，Canvas 为 `integration/newapi-images`。不得将候选状态当上线，不重用旧预算，不重启任何已收口的旧付费验收脚本。

下一步：Owner 批准预算/新运行边界后，在现有生产基础上做最小付费验证，再按已保留配置和镜像安全发布/回滚。主 180 秒、备 240 秒、每款冷却 300 秒、最多两次；Packy low 与真实扣费未知需明确保留。现有全局 RetryTimes=0 和 Canvas 唯一发送保持。

本轮服务器私有工作目录 `/opt/canvas-newapi-autofailover-20261016`；生产镜像未回切，全部本轮测试容器停止。候选镜像不能被旧基础 Compose 静默覆盖；原正式部署/备份/回滚仍见 [上一轮发布回执](CANVAS_IMAGES_20261016.md)。
