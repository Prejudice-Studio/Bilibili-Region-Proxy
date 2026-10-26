# 配置与搜索结果注入字段

配置文件为 UTF-8 JSON。可复制 [`config.example.json`](config.example.json) 后编辑；程序会在后续请求时重新加载配置。

## 顶层字段

| 字段 | JSON 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `$schema` | string | 无 | 编辑器使用的 Schema 文件位置；程序忽略该字段。示例配置指向同目录的 `config.schema.json`。 |
| `force_intl_bangumi_search` | boolean | `false` | 设为 `true` 后，匹配的番剧搜索请求强制使用 `/zh-Hans/bangumi`；不影响普通视频搜索和其他 API。 |
| `injection` | object | 关闭 | 搜索结果注入设置，字段见下表。 |

### `injection`

| 字段 | JSON 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `enabled` | boolean | `false` | 必须为 `true` 才注入。 |
| `position` | string | `tail` | `head` 放在真实结果前，`tail` 放在真实结果后；其他值按 `tail` 处理。 |
| `items` | array of object | `[]` | 注入条目，最多处理前 100 条。每条必须有非空 `title` 和有效链接。 |

注入只作用于第一页的成功番剧搜索响应。注入条目追加到 `data.result` 或兼容的 `data.items`，不增加上游 `data.numResults`。国际网页搜索和普通 HTTP 搜索均支持注入。

## 注入条目字段

类型列表示建议填写的 JSON 类型。实现为兼容旧配置，对部分数值字段也接受数字字符串；表格仍以推荐类型为准。字段名区分大小写。

### 标题、分类、封面与链接

| 字段 | 类型 | 默认/必填 | 说明 |
| --- | --- | --- | --- |
| `title` | string | 必填 | 展示标题；去除首尾空格后不能为空，最多 200 个字符。 |
| `org_title` | string | `""` | 原名字段。`original_title` 是兼容别名。最多 200 字符。 |
| `original_title` | string | `""` | `org_title` 的兼容别名；`org_title` 优先。最多 200 字符。 |
| `type` | string | 自动推断 | 结果类型。通常填写 `media_bangumi`；影视类可填 `media_ft`。未填时根据 `season_type` 推断：类型 1、4 为 `media_bangumi`，其余为 `media_ft`。最多 40 字符。 |
| `image` | string | `""` | 封面地址；优先级高于 `image_url`、`cover`。最多 2048 字符。 |
| `image_url` | string | `""` | `image` 的兼容别名，优先级次于 `image`、高于 `cover`。最多 2048 字符。 |
| `cover` | string | `""` | 封面字段兼容别名。最多 2048 字符。 |
| `link` | string | 必填链接之一 | 跳转地址；链接优先级为 `link`、`url`、`goto_url`、`uri`。 |
| `url` | string | 必填链接之一 | `link` 的兼容别名。 |
| `goto_url` | string | 必填链接之一 | `link` 的兼容别名。 |
| `uri` | string | 必填链接之一 | `link` 的兼容别名。 |
| `param` | string | `""` | 客户端跳转参数，最多 2048 字符。 |
| `goto` | string | `""` | 跳转类型/目标提示，最多 120 字符。 |
| `linktype` | string | `""` | 链接类型提示，最多 120 字符。 |
| `trackid` | string | `""` | 跟踪标识，最多 200 字符。 |
| `target` | string | `""` | 目标字段，最多 2048 字符。 |
| `button_text` | string | `""` | 展示按钮文案，最多 120 字符。 |

链接支持 `https://`、`http://`、`//host/path`、站内 `/path` 和不带协议的域名。`//` 和裸域名会补为 HTTPS，站内路径会补为 `https://www.bilibili.com`。其他协议、无主机名或包含用户名密码的 URL 会导致该条目被跳过。链接最长 2048 字符。

### 地区、类型、状态与季 ID

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `media_type` | integer | `1` | 媒体类型；`0` 会按 `1` 处理。 |
| `areas` | string | `""` | 地区文本，最多 300 字符；`area` 为兼容别名。 |
| `area` | string | `""` | `areas` 的兼容别名。两者同时提供时各自优先作为自身输出字段，否则互相补充。 |
| `styles` | string | `""` | 风格/标签文本，最多 500 字符；`style` 为兼容别名。 |
| `style` | string | `""` | `styles` 的兼容别名。两者同时提供时各自优先作为自身输出字段，否则互相补充。 |
| `season_type` | integer | `1`；`type=media_ft` 时为 `2` | 季度类别编号。常见值：`1` 番剧、`2` 电影、`3` 纪录片、`4` 国创、`5` 电视剧、`7` 综艺。值 `0` 按默认值处理。 |
| `season_type_name` | string | 根据 `season_type` 推断 | 季度类别名称，最多 80 字符。未知编号默认名称为“番剧”。 |
| `season_id` | integer | `0` | 番剧季 ID；`pgc_season_id` 为兼容别名。为 `0` 时不会输出季 ID。若要让客户端使用自定义链接而不是季详情，省略这两个字段或设为 `0`。 |
| `pgc_season_id` | integer | `0` | `season_id` 的兼容别名；`season_id` 优先。 |
| `media_id` | integer | `0` | 媒体 ID；为 `0` 时不输出。 |
| `media_mode` | integer | `0` | 媒体模式编号。 |
| `is_avid` | boolean | `false` | 是否按 avid 类型解释。 |
| `play_state` | integer | `0` | 播放状态编号。 |
| `is_out` | integer | `0` | 是否为站外内容的状态值。 |
| `is_atten` | integer | `0` | 关注状态；该值为空、`0` 或 `false` 时会回退到 `is_follow`。 |
| `is_follow` | integer | `0` | 关注状态字段。也作为 `is_atten` 的回退值。 |
| `is_selection` | integer | `0` | 是否展示选集入口的状态值。 |
| `corner` | integer | `0` | 角标编号。 |
| `angle_title` | string | `""` | 角标文案，最多 200 字符。 |
| `angle_color` | integer | `0` | 角标颜色数值。 |
| `label` | string | `""` | 标签文本，最多 200 字符。 |
| `styles_v2` | string | `""` | 新版风格文本，最多 500 字符。 |
| `selection_style` | string | `""` | 选集展示样式，最多 80 字符。 |

### 评分、简介与播出信息

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `media_score` | object | `null` | 评分对象，包含 `score` 和 `user_count`。只有评分或人数有效时才生成响应中的 `media_score`。 |
| `media_score.score` | number | `0` | 评分数值，例如 `9.2`。 |
| `media_score.user_count` | integer | `0` | 评分人数。 |
| `media_score.score` | number，兼容旧写法 | — | 也可使用条目中的字面键 `"media_score.score"`；优先级低于对象内的 `media_score.score`。 |
| `media_score.user_count` | integer，兼容旧写法 | — | 也可使用字面键 `"media_score.user_count"`；优先级低于对象内的 `media_score.user_count`。 |
| `rating` | number | `0` | `media_score.score` 缺失时的评分回退值；响应中也会输出 `rating`。 |
| `rating_count` | integer | `0` | `media_score.user_count` 缺失时的人数回退值；优先级高于 `vote`。 |
| `vote` | integer | `0` | 评分人数的旧字段回退值。 |
| `desc` | string | `""` | 简介，最多 5000 字符。可由 `description`、`prompt` 回退。 |
| `description` | string | `""` | 简介别名。三个输出字段分别按自身优先级选择：`desc` 优先 `desc`、`description`、`prompt`；`description` 优先 `description`、`desc`、`prompt`；`prompt` 优先 `prompt`、`desc`、`description`。 |
| `prompt` | string | `""` | 简介别名，最多 5000 字符。 |
| `pubtime` | integer | `0` | 发布时间 Unix 时间戳（秒）。 |
| `fix_pubtime_str` | string | `""` | 自定义播出时间文案，最多 200 字符。 |
| `cv` | string | `""` | 配音信息，最多 1000 字符。 |
| `staff` | string | `""` | 制作人员信息，最多 1000 字符。 |
| `hit_epids` | string | `""` | 命中剧集 ID 文本，最多 1000 字符。 |

评分优先级：`media_score.score` → `"media_score.score"` → `rating`；人数优先级：`media_score.user_count` → `"media_score.user_count"` → `rating_count` → `vote`。

### 剧集与列表字段

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `eps` | array | `[]` | 剧集列表，最多 30 项；非空时优先于 `episodes`。列表元素作为 JSON 原样保留。 |
| `episodes` | array | `[]` | `eps` 为空或未设置时作为剧集列表；输出中 `eps` 和 `episodes` 都使用最终选中的列表。 |
| `episodes_new` | array | `[]` | 新版剧集列表，最多 30 项；元素原样保留。 |
| `ep_size` | integer | 最终剧集数量 | 剧集总数/展示数量。 |
| `hit_columns` | array | `[]` | 搜索命中列配置；元素原样保留。 |
| `display_info` | array | `[]` | 展示信息列表；元素原样保留。 |
| `badges` | array | `[]` | 角标列表；元素原样保留。 |
| `badges_v2` | array | `[]` | 新版角标列表；元素原样保留。 |

这些数组只检查顶层是否为 JSON 数组，不会验证或改写元素内部结构。请按目标客户端期望的搜索响应结构填写对象。

### 外部平台与按钮对象

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `all_net` | object | `{}` | 外部平台信息，可包含 `name`、`icon`、`url`，用于补足下方对应字段。 |
| `all_net_name` | string | `all_net.name` | 外部平台名称，最多 200 字符。 |
| `all_net_icon` | string | `all_net.icon` | 外部平台图标 URL，最多 2048 字符。 |
| `all_net_url` | string | `all_net.url` | 外部平台链接，最多 2048 字符。 |
| `out_name` | string | `all_net_name` 或 `all_net.name` | 输出站外来源名称，最多 200 字符。 |
| `out_icon` | string | `all_net_icon` 或 `all_net.icon` | 输出站外来源图标，最多 2048 字符。 |
| `out_url` | string | `all_net_url` 或 `all_net.url` | 输出站外来源链接，最多 2048 字符。 |
| `style_label` | object | `null` | 风格标签结构；对象内容原样保留。 |
| `watch_button` | object | `null` | 观看按钮配置；对象内容原样保留。 |
| `check_more` | object | `null` | “查看更多”按钮配置；对象内容原样保留。 |
| `follow_button` | object | `null` | 追番按钮配置；对象内容原样保留。 |

### 响应自动生成字段

下列字段由代理生成，不需要也不能通过注入配置覆盖：

| 响应字段 | 生成值 |
| --- | --- |
| `resolver_injected` | 固定为 `true`，用于标记该结果由注入产生。 |
| `resolver_injection_source` | 固定为 `Bilibili-Region-Proxy`。 |
| `num_episodes` | 最终剧集数组的长度。 |
| `season_id`、`pgc_season_id` | 从配置 ID 生成；ID 为 `0` 时两个字段都省略。 |
| `media_id` | 从配置生成；值为 `0` 时省略。 |

## 字符串和数值处理

- 字符串字段会去除首尾空白；表格给出的上限按 Unicode 字符数截断。
- 数值字段建议使用 JSON 数字。为兼容旧配置，整数转换也接受数字字符串和布尔值；小数转整数时会截去小数部分。`is_avid` 建议填写 JSON 布尔值；兼容转换按真值处理，非空字符串（包括字符串 `"false"`）会被视为 `true`。
- 数组字段只接受 JSON 数组；对象字段只接受 JSON 对象；不匹配时按空值或默认值处理。
- 条目超过 100 条时，只读取前 100 条；`eps` 和 `episodes_new` 各最多保留 30 项。
- 每条条目独立校验。缺少标题、链接无效或链接过长的条目会被跳过。
