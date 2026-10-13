import { Eye, EyeClosed } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { setValuesHidden, useValuesHidden } from '@/lib/hide-values'

/** Hides or shows every amount in the app (closed eye: hidden). */
export function ValuesToggle() {
  const valuesHidden = useValuesHidden()
  return (
    <Button
      variant="outline"
      size="icon"
      aria-label={valuesHidden ? 'Show values' : 'Hide values'}
      aria-pressed={valuesHidden}
      title={valuesHidden ? 'Show values' : 'Hide values'}
      onClick={() => setValuesHidden(!valuesHidden)}
    >
      {valuesHidden ? <EyeClosed className="size-4" /> : <Eye className="size-4" />}
    </Button>
  )
}
