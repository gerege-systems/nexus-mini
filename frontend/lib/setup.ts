// Анхны тохиргооны шидтэн шаардлагатай юу — landing/login үүнийг асууж /setup
// руу шилжүүлнэ. Кэшлэхгүй: утга амьдралдаа нэг удаа өөрчлөгдөнө. Алдаанд
// false — нэг fetch унасны улмаас нүүр хуудсаа алдахгүй.
import { api } from '@/lib/api';

export type SetupStatus = { required: boolean; armed: boolean };

export async function setupStatus(): Promise<SetupStatus> {
  try {
    return await api.get<SetupStatus>('/api/setup/status');
  } catch {
    return { required: false, armed: false };
  }
}
