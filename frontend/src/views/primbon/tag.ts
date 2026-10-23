import type { Peraturan } from '../../lib/api'
import { pillStatus } from './bantu'

type MetadataTag = Pick<Peraturan, 'status' | 'perubahan' | 'kategori_label' | 'tag'>

export function tagPeraturan(p: MetadataTag) {
  const labels = [p.status?.trim() || 'Status belum diketahui', p.perubahan?.trim(), ...p.kategori_label, ...(p.tag ?? [])]
  const seen = new Set<string>()
  return labels.flatMap((label, i) => {
    if (!label || seen.has(label.toLocaleLowerCase('id-ID'))) return []
    seen.add(label.toLocaleLowerCase('id-ID'))
    return [{ label, warna: i < 2 ? pillStatus(label) : 'pill-status-muted' }]
  })
}
