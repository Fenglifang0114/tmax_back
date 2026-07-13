import os
import re

file_path = r"d:\tmaxbackend\tmaxBack\Backend\svc\srvproto.go"
out_path = r"d:\tmaxbackend\tmaxBack\Backend\docs\ui_service_api.html"

with open(file_path, "r", encoding="utf-8") as f:
    lines = f.readlines()

content = "".join(lines)

# Find all ReqType definitions
req_pattern = re.compile(r'([A-Z0-9_]+)\s+ReqType\s*=\s*"([^"]+)"\s*(?://\s*(.*))?')
reqs = req_pattern.findall(content)

# Map structs
struct_pattern = re.compile(r'type\s+([A-Za-z0-9_]+)\s+struct\s*\{([\s\S]*?)\}')
structs_raw = struct_pattern.findall(content)
structs = {}
for name, body in structs_raw:
    # clean up body, simple presentation
    fields = []
    for line in body.split('\n'):
        line = line.strip()
        if line and not line.startswith('//'):
            parts = line.split()
            if len(parts) >= 2:
                fields.append(f"{parts[0]}: {parts[1]}")
    structs[name] = fields

html = """
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <title>UI 与 Service 通讯接口文档</title>
    <style>
        body { font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif; line-height: 1.6; color: #333; margin: 0; padding: 0; display: flex; }
        .sidebar { width: 300px; background: #f4f6f8; padding: 20px; overflow-y: auto; height: 100vh; position: fixed; }
        .sidebar a { display: block; padding: 5px 0; color: #007bff; text-decoration: none; font-size: 14px; }
        .sidebar a:hover { text-decoration: underline; }
        .content { margin-left: 340px; padding: 40px; max-width: 1200px; width: 100%; }
        h1, h2, h3 { color: #2c3e50; }
        .endpoint { background: #fff; border: 1px solid #ddd; border-radius: 8px; margin-bottom: 20px; box-shadow: 0 2px 4px rgba(0,0,0,0.05); }
        .endpoint-header { background: #f8f9fa; padding: 15px 20px; border-bottom: 1px solid #ddd; display: flex; justify-content: space-between; align-items: center; border-radius: 8px 8px 0 0; }
        .endpoint-title { font-size: 18px; font-weight: bold; color: #e74c3c; }
        .endpoint-name { color: #555; font-size: 14px; }
        .endpoint-body { padding: 20px; }
        pre { background: #2d2d2d; color: #f8f8f2; padding: 15px; border-radius: 5px; overflow-x: auto; font-family: Consolas, monospace; font-size: 14px;}
        .comment { color: #7f8c8d; font-style: italic; margin-bottom: 15px; }
    </style>
</head>
<body>
    <div class="sidebar">
        <h3>接口列表</h3>
"""

for const_name, req_val, comment in reqs:
    html += f'        <a href="#{req_val}">{req_val}</a>\n'

html += """
    </div>
    <div class="content">
        <h1>UI 与 Service (Backend) 接口文档 (完全版)</h1>
        <p>本系统界面 (UI) 与后台服务 (Service) 之间的通信基于 <code>Request</code>。格式为 JSON。协议：<br/> <code>{"Req": "api_name", "ReqData": "json_string"}</code></p>
        <hr>
"""

import json

for const_name, req_val, comment in reqs:
    # Try to extract "with XXX parameter"
    param_type = None
    if "with parameter of" in comment:
        m = re.search(r'with parameter of ([A-Za-z0-9_]+)', comment)
        if m: param_type = m.group(1)
    elif "with " in comment and "parameter" in comment:
        m = re.search(r'with ([A-Za-z0-9_]+)', comment)
        if m: param_type = m.group(1)
        
    html += f'        <div class="endpoint" id="{req_val}">\n'
    html += f'            <div class="endpoint-header">\n'
    html += f'                <div class="endpoint-title">{req_val}</div>\n'
    html += f'                <div class="endpoint-name">{const_name}</div>\n'
    html += f'            </div>\n'
    html += f'            <div class="endpoint-body">\n'
    
    if comment:
        html += f'                <div class="comment">描述/备注：{comment}</div>\n'
    else:
        html += f'                <div class="comment">无备注信息。</div>\n'
        
    if "without parameter" in comment or not param_type:
        html += f'                <p><strong>ReqData 入参:</strong> 无 (空字符串 `""`)</p>\n'
    else:
        html += f'                <p><strong>ReqData 结构:</strong> <code>{param_type}</code></p>\n'
        if param_type in structs:
            html += f'                <pre><code>{{\n'
            for field in structs[param_type]:
                html += f'  "{field.split(":")[0]}": "{field.split(":")[1]}",\n'
            html += f'}}</code></pre>\n'
        else:
            html += f'                <pre><code>// 需参照代码定义: {param_type}</code></pre>\n'
    html += f'            </div>\n'
    html += f'        </div>\n'

html += """
    </div>
</body>
</html>
"""

with open(out_path, "w", encoding="utf-8") as f:
    f.write(html)

print("Documentation generated successfully at", out_path)
