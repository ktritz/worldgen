import struct, gzip, lzma

NA_INT=-2147483648
class P:
    def __init__(s,b,i=0): s.b=b; s.i=i; s.refs=[]
    def u32(s):
        v=struct.unpack('>i',s.b[s.i:s.i+4])[0]; s.i+=4; return v
    def dbl(s):
        v=struct.unpack('>d',s.b[s.i:s.i+8])[0]; s.i+=8; return v
    def raw(s,n):
        v=s.b[s.i:s.i+n]; s.i+=n; return v
    def item(s):
        f=s.u32(); return s.body(f)
    def body(s,f):
        t=f&0xFF; hasat=(f>>9)&1; hastag=(f>>10)&1
        if t==254: return None
        if t==255:
            idx=f>>8
            if idx==0: idx=s.u32()
            return s.refs[idx-1]
        if t in (253,242,241,252,251,250): return '<special%d>'%t
        if t==1:
            v=s.item(); s.refs.append(v); return v
        if t in (2,6,17,239,240):
            out=[]
            cur_f=f
            while True:
                tt=cur_f&0xFF
                if tt==254: break
                at = s.item() if (cur_f>>9)&1 else None
                tag = s.item() if (cur_f>>10)&1 else None
                val = s.item()
                out.append((tag,val))
                cur_f=s.u32()
                if (cur_f&0xFF)==254: break
            return out
        if t==9:
            n=s.u32()
            if n==-1: return None
            return s.raw(n).decode('latin-1')
        if t==10 or t==13:
            n=s.u32(); v=[s.u32() for _ in range(n)]; v=[None if x==NA_INT else x for x in v]; r=v
        elif t==14:
            n=s.u32(); r=[s.dbl() for _ in range(n)]
        elif t==16:
            n=s.u32(); r=[s.item() for _ in range(n)]
        elif t==19:
            n=s.u32(); r=[s.item() for _ in range(n)]
        elif t==24:
            n=s.u32(); r=s.raw(n)
        else:
            raise Exception('type %d @%d'%(t,s.i))
        obj={'_v':r}
        if hasat:
            a=s.item()
            obj['_a']={k:v for k,v in a}
        return obj

def unwrap(o):
    return o['_v'] if isinstance(o,dict) else o

def to_df(o):
    a=o.get('_a',{})
    names=unwrap(a['names'])
    cols=o['_v']
    out={}
    for n,c in zip(names,cols):
        ca = c.get('_a',{}) if isinstance(c,dict) else {}
        vals=unwrap(c)
        if 'levels' in ca:
            lev=unwrap(ca['levels'])
            vals=[None if v is None else lev[v-1] for v in vals]
        out[n]=vals
    return out

def load_rds(path):
    b=open(path,'rb').read()
    if b[:2]==b'\x1f\x8b': b=gzip.open(path,'rb').read()
    elif b[:2]==b'\xfd7': b=lzma.open(path,'rb').read()
    return load_bytes(b)

def load_bytes(b):
    i=0
    if b[:5]==b'RDX2\n': i=5
    assert b[i:i+2]==b'X\n'; i+=2
    p=P(b,i)
    ver=p.u32(); p.u32(); p.u32()
    if ver>=3:
        n=p.u32(); p.raw(n)
    return p.item()
