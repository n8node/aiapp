from __future__ import annotations

import os
import tempfile
import unittest

from app.services.extract import extract_path


class ExtractTests(unittest.TestCase):
    def test_txt(self) -> None:
        with tempfile.NamedTemporaryFile(suffix=".txt", delete=False) as tmp:
            tmp.write("привет мир".encode("utf-8"))
            path = tmp.name
        try:
            out = extract_path(path, "note.txt")
        finally:
            os.unlink(path)
        self.assertEqual(out.engine, "text")
        self.assertIn("привет", out.text)

    def test_image_ocr_required(self) -> None:
        out = extract_path("unused", "scan.png")
        self.assertEqual(out.engine, "none")
        self.assertEqual(out.text, "")
        self.assertIn("ocr_required", out.warnings)

    def test_unsupported(self) -> None:
        out = extract_path("unused", "clip.mp4")
        self.assertIn("unsupported_type", out.warnings)


if __name__ == "__main__":
    unittest.main()
