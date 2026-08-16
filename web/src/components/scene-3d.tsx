import { useEffect, useRef } from 'react'
import * as THREE from 'three'

// 顺滑的三维背景：多层低频 fbm 流动，柔和的单色/微冷色渐变。
// 用于仪表盘等页面背景；标签页隐藏时自动暂停以节省性能。
const VERT = `
varying vec2 vUv;
void main() {
  vUv = uv;
  gl_Position = vec4(position, 1.0);
}
`

const FRAG = `
uniform float uTime;
uniform float uOpacity;
varying vec2 vUv;

float hash(vec2 p) {
  return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453123);
}

float noise(vec2 p) {
  vec2 i = floor(p);
  vec2 f = fract(p);
  f = f * f * (3.0 - 2.0 * f);
  float a = hash(i);
  float b = hash(i + vec2(1.0, 0.0));
  float c = hash(i + vec2(0.0, 1.0));
  float d = hash(i + vec2(1.0, 1.0));
  return mix(mix(a, b, f.x), mix(c, d, f.x), f.y);
}

float fbm(vec2 p) {
  float v = 0.0;
  float a = 0.5;
  for (int i = 0; i < 5; i++) {
    v += a * noise(p);
    p *= 2.03;
    a *= 0.5;
  }
  return v;
}

void main() {
  vec2 uv = vUv;
  vec2 p = (uv - 0.5) * vec2(1.0, 0.6);
  float t = uTime * 0.06;

  float n1 = fbm(p * 2.1 + vec2(t, t * 0.7));
  float n2 = fbm(p * 3.0 - vec2(t * 0.8, t * 0.5) + 5.2);
  float n3 = fbm(p * 1.5 + vec2(-t * 0.6, t * 0.9) + 9.1);

  float v = n1 * 0.55 + n2 * 0.3 + n3 * 0.25;
  v = smoothstep(0.12, 0.88, v);

  float g = mix(0.08, 0.32, v);
  vec3 col = vec3(g);
  col.b += (n2 - 0.5) * 0.10;
  col.r += (n3 - 0.5) * 0.06;

  float alpha = uOpacity * (0.32 + 0.42 * v);
  gl_FragColor = vec4(col, alpha);
}
`

export function Scene3D({ className }: { className?: string }) {
  const mountRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    const mount = mountRef.current
    if (!mount) return

    const renderer = new THREE.WebGLRenderer({ alpha: true, antialias: true, powerPreference: 'high-performance' })
    renderer.setClearColor(0x000000, 0)
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 1.75))
    mount.appendChild(renderer.domElement)

    const scene = new THREE.Scene()
    const camera = new THREE.OrthographicCamera(-1, 1, 1, -1, 0, 1)

    const geometry = new THREE.PlaneGeometry(2, 2)
    const material = new THREE.ShaderMaterial({
      vertexShader: VERT,
      fragmentShader: FRAG,
      uniforms: { uTime: { value: 0 }, uOpacity: { value: 1 } },
      transparent: true,
      depthWrite: false
    })
    scene.add(new THREE.Mesh(geometry, material))

    const resize = () => {
      const w = mount.clientWidth || 1
      const h = mount.clientHeight || 1
      renderer.setSize(w, h)
    }
    resize()

    const clock = new THREE.Clock()
    let frame = 0
    let running = true

    const animate = () => {
      frame = requestAnimationFrame(animate)
      if (running) {
        material.uniforms.uTime.value = clock.getElapsedTime()
      }
      renderer.render(scene, camera)
    }
    animate()

    const onVisibility = () => {
      running = !document.hidden
      if (running) clock.getDelta() // 跳过隐藏期间的累计时间
    }
    document.addEventListener('visibilitychange', onVisibility)

    const observer = new ResizeObserver(resize)
    observer.observe(mount)

    return () => {
      cancelAnimationFrame(frame)
      document.removeEventListener('visibilitychange', onVisibility)
      observer.disconnect()
      geometry.dispose()
      material.dispose()
      renderer.dispose()
      if (renderer.domElement.parentNode === mount) {
        mount.removeChild(renderer.domElement)
      }
    }
  }, [])

  return <div ref={mountRef} className={className} aria-hidden="true" />
}
