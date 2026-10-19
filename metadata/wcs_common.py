import json, os, re, time, hashlib, urllib.parse
from pathlib import Path
import requests
import packaging.version as pv
S=requests.Session(); S.headers.update({'User-Agent':'wcs-index-bot/2.0 (+wcs-repos full index)'})
ROOT=Path('/mnt/agents/output/wcs-repos')
OUT=Path('/tmp/wcs_all'); OUT.mkdir(exist_ok=True)

def get(url, **kw):
    last=None
    for i in range(4):
        try:
            kw.setdefault('timeout',(20,80)); r=S.get(url,**kw); r.raise_for_status(); return r
        except Exception as e:
            last=e; time.sleep(1+i)
    raise last

def sha256_stream(url):
    h=hashlib.sha256()
    with S.get(url,stream=True,timeout=(20,180)) as r:
        r.raise_for_status()
        for chunk in r.iter_content(1024*1024):
            if chunk: h.update(chunk)
    return h.hexdigest()

def load(name, default=None):
    p=OUT/name
    if p.exists(): return json.loads(p.read_text())
    return default

def save(name,obj):
    (OUT/name).write_text(json.dumps(obj,ensure_ascii=False,indent=2)+'\n')

def ver_key(v):
    s=str(v).lstrip('v')
    nums=[int(x) for x in re.findall(r'\d+',s.split('-')[0].split('+')[0])]
    while len(nums)<4: nums.append(0)
    # prerelease/qualifier 排后；对 2016+ 语言版本足够稳定
    qual=1 if re.search(r'(alpha|beta|rc|preview|next|dev)',s,re.I) else 0
    return (nums[0],nums[1],nums[2],nums[3],qual,s)
def xy(v):
    s=str(v).lstrip('v').split('-')[0].split('+')[0]
    p=re.findall(r'\d+',s)
    return '.'.join(p[:2]) if len(p)>=2 else s

def select_latest_per_xy(items, version_getter, date_getter=None, min_version=None, min_date='2016-01-01'):
    groups={}
    for it in items:
        v=str(version_getter(it)).lstrip('v')
        try: vk=ver_key(v)
        except Exception: continue
        if min_version and vk < ver_key(min_version): continue
        k=xy(v)
        cur=groups.get(k)
        better=False
        if cur is None: better=True
        else:
            cv=version_getter(cur); ck=ver_key(str(cv).lstrip('v'))
            if vk>ck: better=True
            elif vk==ck and date_getter:
                dv=date_getter(it); dc=date_getter(cur)
                if dv and dc and dv>dc: better=True
        if better: groups[k]=it
    out=[]
    for it in groups.values():
        d=date_getter(it) if date_getter else None
        v=str(version_getter(it)).lstrip('v')
        # 以该 x.y 线最新 patch 的发布日期判断是否进入 2016+；跨 2016 的线默认包含
        if (d is None or d>=min_date) and ver_key(v) >= ver_key(min_version or '0'):
            out.append(it)
    return sorted(out,key=lambda it: ver_key(str(version_getter(it)).lstrip('v')))

def artifact(version, filename, url, sha256, **metadata):
    result = {'version': str(version), 'filename': filename, 'url': url, 'sha256': sha256}
    if metadata:
        result['metadata'] = metadata
    return result


def vindex(software, version, packages, **metadata):
    if not isinstance(packages, dict) or not packages:
        raise ValueError('v4 packages must be a nonempty mapping keyed by package name')
    result = {'schema_version': 4, 'software': software, 'version': str(version), 'packages': packages}
    if metadata:
        result['metadata'] = metadata
    return result


def top_index(software, versions, **metadata):
    result = {'schema_version': 4, 'software': software,
              'versions': [{'version': str(v), 'path': f'{v}/index.json'} for v in versions]}
    if metadata:
        result['metadata'] = metadata
    return result


def write_json(rel,obj):
    p=ROOT/rel; p.parent.mkdir(parents=True,exist_ok=True)
    tmp=p.with_suffix(p.suffix+'.tmp'); tmp.write_text(json.dumps(obj,ensure_ascii=False,indent=2)+'\n'); os.replace(tmp,p)
