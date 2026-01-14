#!/usr/bin/env python3
"""
Scraper wrapper that outputs JSON for Go worker integration.
This wraps the ground-reservation scrapers and outputs structured JSON.
"""

import json
import os
import re
import sys
from datetime import datetime
from typing import Dict, Any, List, Optional

# Configure all scrapers with wide time range (00:00 - 23:59) and all weekdays
ALL_WEEKDAYS = "月曜日,火曜日,水曜日,木曜日,金曜日,土曜日,日曜日,祝日"
for prefix in ("HIRATSUKA", "AYASE", "YOKOHAMA"):
    os.environ.setdefault(f"{prefix}_TIME_FROM", "0000")
    os.environ.setdefault(f"{prefix}_TIME_TO", "2359")
    os.environ.setdefault(f"{prefix}_SELECTED_WEEK_DAYS", ALL_WEEKDAYS)

script_dir = os.path.dirname(os.path.abspath(__file__))
ground_reservation_path = os.path.join(script_dir, '..', '..', 'ground-reservation')
sys.path.insert(0, ground_reservation_path)

try:
    from app.facilities.yokohama.yokohama_facility import YokohamaFacility
    from app.facilities.ayase.ayase_facility import AyaseFacility
    from app.facilities.hiratsuka.hiratsuka_facility import HiratsukaFacility
    from app.facilities.kanagawa_system.kanagawa.kanagawa_facility import KanagawaFacility
    from app.facilities.kanagawa_system.kamakura.kamakura_facility import KamakuraFacility
    from app.facilities.kanagawa_system.fujisawa.fujisawa_facility import FujisawaFacility

    SCRAPERS_AVAILABLE = True
except ImportError as e:
    SCRAPERS_AVAILABLE = False
    IMPORT_ERROR = str(e)


# Diagnostic status codes
class ScrapeStatus:
    SUCCESS = "success"                    # スクレイピング成功、結果あり
    SUCCESS_NO_SLOTS = "success_no_slots"  # スクレイピング成功、空き枠なし
    PARSE_ERROR = "parse_error"            # HTMLの解析に失敗
    NETWORK_ERROR = "network_error"        # ネットワークエラー
    NO_TABLE_FOUND = "no_table_found"      # 検索結果テーブルが見つからない
    UNKNOWN_FACILITY = "unknown_facility"  # 不明な施設タイプ
    SCRAPER_UNAVAILABLE = "scraper_unavailable"  # スクレイパーが利用不可


def _make_slot(date: str, time_from: str, time_to: str, court_name: str,
                raw_text: str, facility_type: str) -> Dict[str, Any]:
    """Create a slot dict with parsed values."""
    return {
        "date": date,
        "time_from": time_from,
        "time_to": time_to,
        "court_name": court_name.strip() if court_name else None,
        "raw_text": raw_text,
        "facility_type": facility_type
    }


def _infer_year(month: int) -> int:
    """Infer year from month - if month is past, assume next year."""
    now = datetime.now()
    return now.year + 1 if month < now.month else now.year


def parse_slot_string(slot_str: str, facility_type: str) -> Dict[str, Any]:
    """Parse slot string into structured dict. Supports multiple date formats."""
    slot_str = slot_str.strip()

    # Japanese era format: "令和08年02月28日(土) 06:00 ～ 08:00 施設名"
    if match := re.match(r'令和(\d+)年(\d+)月(\d+)日\([^)]+\)\s+(\d+:\d+)\s*[～~-]\s*(\d+:\d+)\s+(.+)', slot_str):
        era_year, month, day, time_from, time_to, court_name = match.groups()
        year = 2018 + int(era_year)  # 令和元年 = 2019
        return _make_slot(f"{year}-{int(month):02d}-{int(day):02d}",
                          time_from, time_to, court_name, slot_str, facility_type)

    # Slash format: "01/17(土) 13:00 ～ 15:00 施設名"
    if match := re.match(r'(\d+)/(\d+)\([^)]+\)\s+(\d+:\d+)\s*[～~-]\s*(\d+:\d+)\s+(.+)', slot_str):
        month, day, time_from, time_to, court_name = match.groups()
        year = _infer_year(int(month))
        return _make_slot(f"{year}-{int(month):02d}-{int(day):02d}",
                          time_from, time_to, court_name, slot_str, facility_type)

    # Kanji format: "1月15日(土) 09:00-12:00 施設名"
    if match := re.match(r'(\d+)月(\d+)日\([^)]+\)\s+(\d+:\d+)\s*[～~-]\s*(\d+:\d+)\s+(.+)', slot_str):
        month, day, time_from, time_to, court_name = match.groups()
        year = _infer_year(int(month))
        return _make_slot(f"{year}-{int(month):02d}-{int(day):02d}",
                          time_from, time_to, court_name, slot_str, facility_type)

    # ISO format: "2024-01-15 09:00-12:00 施設名"
    if match := re.match(r'(\d{4}-\d{2}-\d{2})\s+(\d+:\d+)-(\d+:\d+)\s+(.+)', slot_str):
        date, time_from, time_to, court_name = match.groups()
        return _make_slot(date, time_from, time_to, court_name, slot_str, facility_type)

    # Fallback: unparseable
    return _make_slot(None, None, None, None, slot_str, facility_type)


def create_result(
    facility_type: str,
    status: str,
    slots: Optional[List[Dict[str, Any]]] = None,
    error: Optional[str] = None,
    diagnostics: Optional[Dict[str, Any]] = None
) -> Dict[str, Any]:
    """
    Create a standardized result dict.
    success=True only when status is "success" or "success_no_slots"
    """
    success_statuses = {ScrapeStatus.SUCCESS, ScrapeStatus.SUCCESS_NO_SLOTS}
    return {
        "success": status in success_statuses,
        "status": status,
        "error": error,
        "facility_type": facility_type,
        "slots": slots or [],
        "diagnostics": diagnostics or {},
        "scraped_at": datetime.now().isoformat()
    }


def verify_yokohama_html(facility) -> Dict[str, Any]:
    """
    横浜市のHTMLレスポンスを検証して診断情報を返す
    """
    try:
        from bs4 import BeautifulSoup
        
        search_dates = facility.get_search_dates(0)
        token = facility.get_token(facility.facility_urls['login_page_url'])
        payload = facility.create_payload(token, search_dates)
        facility.send_search_facility_query(facility.facility_urls['facility_search_url'], payload)
        html = facility.get_available_facilities(facility.facility_urls['facility_status_url'])
        
        if not html:
            return {"verified": False, "reason": "HTML response is empty"}
        
        soup = BeautifulSoup(html, 'html.parser')
        
        # 検索結果のテーブルを探す
        table = soup.find('table', class_='table table-bordered table-striped facilities')
        
        # 「空き施設がN件見つかりました」メッセージを探す
        page_text = soup.get_text()
        found_msg = None
        if '空き施設が' in page_text and '件見つかりました' in page_text:
            import re
            match = re.search(r'空き施設が(\d+)件見つかりました', page_text)
            if match:
                found_msg = f"空き施設が{match.group(1)}件見つかりました"
        
        if '条件に該当する施設はありません' in page_text:
            return {
                "verified": True,
                "has_table": False,
                "server_message": "条件に該当する施設はありません",
                "confirmation": "サーバーからの明示的な回答: 空き枠なし"
            }
        
        if found_msg and '0件' in found_msg:
            return {
                "verified": True, 
                "has_table": table is not None,
                "server_message": found_msg,
                "confirmation": "サーバーからの明示的な回答: 空き枠0件"
            }
        
        return {
            "verified": True,
            "has_table": table is not None,
            "server_message": found_msg,
            "html_length": len(html)
        }
    except Exception as e:
        return {"verified": False, "reason": str(e)}


def search_facility(facility_type: str) -> Dict[str, Any]:
    """
    Search a specific facility and return structured results with diagnostics.
    
    Returns a result dict with:
    - success: bool - True if scraping completed (even with 0 results)
    - status: str - Detailed status code for debugging
    - error: str|None - Error message if failed
    - slots: list - Found slots (can be empty)
    - diagnostics: dict - Additional debug info (search dates, HTTP status, etc.)
    """
    global IMPORT_ERROR
    if not SCRAPERS_AVAILABLE:
        return create_result(
            facility_type,
            ScrapeStatus.SCRAPER_UNAVAILABLE,
            error=f"Scrapers not available: {IMPORT_ERROR if 'IMPORT_ERROR' in globals() else 'unknown error'}"
        )
    
    scrapers = {
        "yokohama": YokohamaFacility,
        "ayase": AyaseFacility,
        "hiratsuka": HiratsukaFacility,
        "kanagawa": KanagawaFacility,
        "kamakura": KamakuraFacility,
        "fujisawa": FujisawaFacility,
    }
    
    if facility_type not in scrapers:
        return create_result(
            facility_type,
            ScrapeStatus.UNKNOWN_FACILITY,
            error=f"Unknown facility type: {facility_type}. Available: {list(scrapers.keys())}"
        )
    
    diagnostics = {
        "search_started": datetime.now().isoformat(),
    }
    
    try:
        facility = scrapers[facility_type]()
        raw_results = facility.search_facility()
        
        diagnostics["raw_results_count"] = len(raw_results) if raw_results else 0
        diagnostics["search_completed"] = datetime.now().isoformat()
        
        # Parse slots
        slots = []
        parse_errors = 0
        for slot_str in (raw_results or []):
            if slot_str:  # Skip empty strings
                parsed = parse_slot_string(slot_str, facility_type)
                slots.append(parsed)
                if parsed.get("date") is None:
                    parse_errors += 1
        
        diagnostics["parsed_slots_count"] = len(slots)
        diagnostics["parse_errors"] = parse_errors
        
        # Determine status based on results
        if len(slots) > 0:
            status = ScrapeStatus.SUCCESS
        else:
            # Scraping succeeded but no slots found
            # 横浜の場合、追加の検証を実行
            if facility_type == "yokohama":
                verification = verify_yokohama_html(facility)
                diagnostics["html_verification"] = verification
                if verification.get("confirmation"):
                    diagnostics["note"] = verification["confirmation"]
                else:
                    diagnostics["note"] = "検索は完了しましたが、空き枠は見つかりませんでした。"
            else:
                diagnostics["note"] = "検索は完了しましたが、空き枠は見つかりませんでした。"
            
            status = ScrapeStatus.SUCCESS_NO_SLOTS
        
        return create_result(facility_type, status, slots=slots, diagnostics=diagnostics)
        
    except ConnectionError as e:
        return create_result(
            facility_type,
            ScrapeStatus.NETWORK_ERROR,
            error=str(e),
            diagnostics=diagnostics
        )
    except Exception as e:
        # Capture exception type for better debugging
        diagnostics["exception_type"] = type(e).__name__
        return create_result(
            facility_type,
            ScrapeStatus.PARSE_ERROR,
            error=str(e),
            diagnostics=diagnostics
        )


def main():
    if len(sys.argv) < 2:
        print(json.dumps({
            "success": False,
            "error": "Usage: scraper_wrapper.py <facility_type>",
            "available_types": ["yokohama", "ayase", "hiratsuka", "kanagawa", "kamakura", "fujisawa"]
        }))
        sys.exit(1)
    
    facility_type = sys.argv[1]
    result = search_facility(facility_type)
    print(json.dumps(result, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
