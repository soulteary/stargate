"""Palette expectations must follow the theme rather than a particular design."""
import unittest

from theme import background_for_mode


class ThemePalette(unittest.TestCase):
    def test_base_and_modes(self):
        css = ('.auth-themed{--sg-background:#abcdef;--sg-text:#111111;}'
               '@media(prefers-color-scheme:dark){.auth-themed{--sg-background:#102030;}}')
        self.assertEqual(background_for_mode(css, 'light'), 'rgb(171, 205, 239)')
        self.assertEqual(background_for_mode(css, 'dark'), 'rgb(16, 32, 48)')
        self.assertEqual(background_for_mode('.auth-themed{--sg-background:#ffffff;}', 'dark'), 'rgb(255, 255, 255)')

    def test_missing_palette_is_an_error(self):
        for css in ('', '.auth-themed{--sg-text:#111111;}'):
            with self.subTest(css=css), self.assertRaises(ValueError):
                background_for_mode(css, 'light')
