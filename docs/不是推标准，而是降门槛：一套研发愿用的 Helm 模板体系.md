一、背景：GitOps交付体系下的新挑战
    在当前的系统交付体系中，我们全面引入了 GitOps 模式（见神灯文章： 用 GitOps 重塑 ToB 运维交付链条 ），通过 Git 仓库统一管理部署清单，借助 ArgoCD 实现自动化部署与回滚，底层资源编排则依赖 Helm Chart 进行完成。

这种模式在可审计性、可追溯性、自动化水平上带来了显著提升，但也暴露出一个关键问题：研发团队对 Helm Chart 的掌握程度不足。

尽管 Helm 极大简化了 Kubernetes 资源管理，但对大多数研发人员而言，依然存在较高的学习门槛：

◦values.yaml 配置结构复杂，难以记忆；
◦模板语法（如 {{- if }}、{{ .Values.xxx }}）理解成本高；
◦进阶特性如依赖（dependencies）、钩子（hooks）、条件渲染（conditionals）使用困难；
◦新增服务时需要频繁复制粘贴 Chart 包，容易出错且维护负担重。
在此背景下，提升研发在 GitOps 体系下的交付体验，降低应用交付的复杂度，成为我必须优先解决的问题。


二、目标设定：做一套“让研发敢用、愿用、好用”的Chart模板体系
我设定了几个清晰的目标：

◦极简化：让研发只需关注应用自身的少量必要配置，隐藏复杂细节；
◦规范化：所有应用Chart包结构统一，符合最佳实践，降低维护和运维风险；
◦模块化：通过dependencies引用公共模块，支持按需组合，避免重复造轮子；
◦易扩展：未来新增通用特性（如统一探活机制、统一配置注入、统一暴露方式）时，不需要手动修改各应用Chart，只需在公共模块中演进。
最终，我希望研发写一个新的应用Chart包时，像写配置文件一样简单，而不是“苦学Helm模板语法”。

三、方案设计：搭建Chart模板体系
为了达成上述目标，我从以下几个方面着手设计：

![image-20260728095701241](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728095701241.png)

﻿﻿﻿

3.1. 拆分公共模块（base charts）
首先，我将各种通用的Kubernetes资源抽象成了独立的模块，每个模块做成了一个子Chart，放在统一的公共仓库中，主要包括：

◦基础部署（deployment-base）：标准的Deployment模板，内置标准探活配置（startupProbe、livenessProbe、readinessProbe）；
◦服务暴露（service-base）：标准Service模板，支持ClusterIP、NodePort、LoadBalancer、Ingress；
◦配置管理（config-base）：标准的ConfigMap、Secret模板；
◦动态扩缩容（hpa-base）：标准的动态扩缩容模版；
◦存储管理（pvc-base）：标准的存储pvc模版。
 ﻿  ![image-20260728095716761](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728095716761.png)

﻿

3.2. 提供应用层模板（app chart）
在应用层，我又抽象出了一层统一的Application Chart，研发在新增应用时，只需基于这个Application Chart进行轻量级开发：

◦只需要维护一个简洁的values.yaml文件；
◦在Chart.yaml中通过dependencies引用公共模块；
◦通过values.yaml传参即可完成定制，例如是否开启探活、服务暴露端口、指定挂载配置等等。
例如，一个最简单的values.yaml可以长这样：

```yaml
replicaCount: 2

image:
  repository: harbor-internal.jdt.com.cn/jdx-ka-v2/xxxx
  pullPolicy: IfNotPresent
  tag: "main-1"

service:
  type: ClusterIP
  port: 8080
      
configsPath: /export/servers/configs

fileConfigs:

  # 启动脚本start.sh名字不要变

  # 下面定义start.sh脚本内容，这里可以随意定义启动逻辑，例如先拷贝自定义配置文件后在进行启动。

  start.sh:
    mountPath: "/export/servers/start.sh"
    fileData: |-
      #! /bin/sh
      xxx
```


研发只需关注镜像、环境变量、端口配置，底层资源由模板自动生成。

![image-20260728095835564](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728095835564.png)

 ﻿  

﻿

3.3. 配置文件注入
通过标准化的 Chart 模板，配置注入变得更加灵活且简洁，目前支持三种注入方式。

方法一：Chart目录下文件注入 （⭐️推荐⭐️）
每个应用Chart包都默认会存在config-files 目录，该目录下的文件会自动映射至容器指定目录，默认为/export/servers/configs 该目录修改方法：values.yaml => configsPath: /export/servers/configs
第一步：创建配置文件
 ﻿  ![image-20260728095853109](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728095853109.png)

﻿

第二步：同步（自动模式下无需同步）

![image-20260728095905730](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728095905730.png) ﻿  

第三步：登录容器查看效果
 ﻿  ![image-20260728095918712](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728095918712.png)

﻿

方法二：配置文件注入

```yaml
修改chart包的values.yaml文件即可

### 以配置文件的方式注入至容器 ###

# 启动脚本 对应的文件

configStartAppScriptPath: /export/servers/start.sh
configsPath: /export/servers/configs

# 声明每个配置文件的内容

fileConfigs:

  # 启动脚本start.sh名字不要变

  # 下面定义start.sh脚本内容，这里可以随意定义启动逻辑，例如先拷贝自定义配置文件后在进行启动。

  start.sh: |-
    #!/bin/bash
    

    # 启动程序
    /export/servers/jdk/bin/java ${PFINDER_AGENT:-} -server $JAVA_OPTS -cp "/export/servers/app:/export/servers/app/config:/export/servers/app/lib/*" com.jd.jnos.gateway.Application

  # 自定义文件名file1，会自动映射到容器/export/servers/configs/file1

  file1: |-
    这里是file1文件内容

  # 自定义文件名test，会自动映射到容器/export/servers/configs/test1

  test: |-
    这里是test文件内容

  # 无限增加文件

  application.properties: |-
    这里是test文件内容

  # 无限增加文件
```



方法三： 环境变量注入
Spring Boot 配置文件读取系统环境变量

规则

1.使用下划线_代替点.

2.删除中划线-

3.转为大写

使用大写字母、数字、下划线组成的键，可以读取系统环境变量。

示例

spring.main.log-startup-info转为SPRING_MAIN_LOGSTARTUPINFO

默认值

使用:分隔，冒号后面的是默认值。

ENV_KEY_1: ${JAVA_HOME}
ENV_KEY_2: ${MY_ENV_2:this is default value env2}
ENV_KEY_3: ${MY_ENV_3:this is default value env3}


3.4. 多项目管理模式（项目管理）
考虑到不通的产品有不同环境（dev/test/prod）需要不同配置，我进一步规范了Chart包的组织方式，按如下分层：

 ﻿  ![image-20260728100016276](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728100016276.png)

﻿

并且在Koala（ArgoCD）的配置中，支持不同环境自动引用不同的Chart包。

 ﻿  ![image-20260728100044775](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728100044775.png)

﻿

3.5. 提供完整示例与最佳实践文档
为了让研发快速上手，我还同步做了：

◦一个可以直接复制的应用Chart示例项目；
◦图文并茂的使用手册，教研发如何继承Application Chart，如何填写values.yaml；
◦常见问题（FAQ），比如：如何增加env变量？如何新增volume？如何关闭探活？
﻿
四、配置全局固化
1. 背景


﻿![image-20260728100058339](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728100058339.png)

通过集中管理公共配置（setup Chart），提升代码复用性、降低维护成本，增强交付一致性。

2. 使用指南
第一步：
创建setup配置chart包并增加需要固化的配置：

 ﻿  ![image-20260728100131749](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728100131749.png)

﻿

配置详细内容：

```yaml
apiVersion: v2
name: setup
description: JDX KA-V2-SYSTEM ENV-SETUP
icon: https://storage.360buyimg.com/portalfile/icon/retaillogo.ico
type: application

# 应用Chart包版本

version: 1.0.0

# 应用代码版本

appVersion: "1.0.0"

maintainers:

  - email: zongzhuangkai@jd.com
    name: Zhuangkai Zong
```



  - 
    第二步：
请确认koala-library-chart版本大于1.3.2，所有应用增加如下依赖。
例如：b2c-core-finance应用，修改该应用Chart包中的Chart.yaml：

```yaml
dependencies:

  - name: koala-library-chart
    version: 1.3.2
    repository: "oci://harbor-internal.jdt.com.cn/koala-charts/library"
  - name: koala-common
    version: 2.19.2-1
    repository: "oci://harbor-internal.jdt.com.cn/koala-charts/library"
  - name: setup
    version: 1.0.0
    repository: "file://../../setup"
```

![image-20260728100533051](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728100533051.png)


﻿

第三步：
应用配置文件调用setup固化配置：

注意：配置文件所有的{{ xxxx }} 都将会被解析，应用配置中禁止使用该方式，同时使用会有冲突，同步会失败，如果必须使用请关闭集中固化配置，方法：去除koala-common、setup 依赖包

```yaml
spring.redis.enabled=true
spring.redis.host={{ .Values.setup.domain.finance.redis.host }}
spring.redis.port={{ .Values.setup.domain.finance.redis.port }}
spring.redis.password={{ .Values.setup.domain.finance.redis.password }}
```

﻿![image-20260728100616081](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728100616081.png)

第四步（自动同步忽略）：
1) 同步配置（发布）

 ﻿  

﻿![image-20260728100626239](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728100626239.png)

2) 登录容器查看配置文件（验证）

![image-20260728100635212](C:\Users\王\AppData\Roaming\Typora\typora-user-images\image-20260728100635212.png)

 ﻿  


五、实际效果：交付提效与研发体验双提升
通过这套Chart模板体系的建设，带来的实际效果非常明显：

项目	变化前	变化后
新增一个应用Chart包	平均需要1-2小时，需大量复制修改	1分钟
研发常见提问数量	每周10+次	大幅下降，几乎不再提问Helm细节问题
配置错误率	高（容易出错）	低（自动生成资源规范化）
交付一致性	低，各自为政	高，所有应用遵循统一模板
最重要的是，研发不再畏惧Helm，而是能像写参数表一样，轻松编写和维护自己的应用Chart。


六、心得与反思
这次模板化体系建设让我深刻体会到：

◦纯粹推行标准，是很难让人自发接受的，只有降低使用门槛，才能真正推动标准落地；
◦抽象要适度，既要隐藏复杂性，也要保留灵活性，否则容易出现“定死无法扩展”的问题；
◦文档、示例项目和答疑支持，是体系成功落地不可或缺的一环。
Helm本身作为一款强大的工具，需要正确地引导和包装，才能真正服务好研发和交付体系。

而通过这套模板化的实践，我们用工程化思维，让Helm Chart变得不再复杂，从而让研发专注于最重要的事情——构建他们的应用本身。